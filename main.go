package main

import (
	"database/sql"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cdzombak/image-analyzer-go"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/image/webp"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
	} `yaml:"database"`
	GrayscaleTolerance float64 `yaml:"grayscale_tolerance"`
	ImageBaseURL       string `yaml:"image_base_url"`
}

// App represents the application state
type App struct {
	config   *Config
	db       *sql.DB
	verbose  bool
	logger   *log.Logger
	bwTagID  int64
}

func main() {
	var configFile = flag.String("config", "./config.yml", "Path to configuration file")
	var verbose = flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	app := &App{
		verbose: *verbose,
		logger:  log.New(os.Stdout, "[lychee-bw-tagger] ", log.LstdFlags),
	}

	// Load configuration
	if err := app.loadConfig(*configFile); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	if err := app.connectDatabase(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer app.db.Close()

	// Ensure database schema is ready
	if err := app.prepareDatabase(); err != nil {
		log.Fatalf("Failed to prepare database: %v", err)
	}

	// Find or create Black & White tag
	if err := app.findOrCreateBWTag(); err != nil {
		log.Fatalf("Failed to find/create Black & White tag: %v", err)
	}

	// Process photos
	if err := app.processPhotos(); err != nil {
		log.Fatalf("Failed to process photos: %v", err)
	}

	app.logger.Println("Processing completed successfully")
}

// loadConfig loads configuration from YAML file
func (app *App) loadConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{
		GrayscaleTolerance: 0.1, // default tolerance
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.Port == 0 {
		config.Database.Port = 3306 // default MySQL port
	}
	if config.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.ImageBaseURL == "" {
		return fmt.Errorf("image_base_url is required")
	}

	app.config = config
	return nil
}

// connectDatabase establishes connection to MySQL database
func (app *App) connectDatabase() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local",
		app.config.Database.Username,
		app.config.Database.Password,
		app.config.Database.Host,
		app.config.Database.Port,
		app.config.Database.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	app.db = db
	app.logger.Println("Successfully connected to database")
	return nil
}

// prepareDatabase ensures the database schema is ready
func (app *App) prepareDatabase() error {
	// Add _dz_bw column if it doesn't exist
	query := `
		ALTER TABLE photos
		ADD COLUMN IF NOT EXISTS _dz_bw TINYINT(1) NULL COMMENT 'Black & white detection result'`

	if _, err := app.db.Exec(query); err != nil {
		return fmt.Errorf("failed to add _dz_bw column: %w", err)
	}

	app.logger.Println("Database schema is ready")
	return nil
}

// findOrCreateBWTag finds the Black & White tag or creates it if it doesn't exist
func (app *App) findOrCreateBWTag() error {
	const bwTagName = "Black & White"

	// Try to find existing tag
	var tagID int64
	err := app.db.QueryRow("SELECT id FROM tags WHERE name = ?", bwTagName).Scan(&tagID)
	if err == nil {
		app.bwTagID = tagID
		app.logger.Printf("Found existing Black & White tag with ID: %d", tagID)
		return nil
	}

	// Tag not found, create it
	result, err := app.db.Exec("INSERT INTO tags (name, description) VALUES (?, ?)",
		bwTagName, "Automatically detected black and white photos")
	if err != nil {
		return fmt.Errorf("failed to create Black & White tag: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get new tag ID: %w", err)
	}

	app.bwTagID = id
	app.logger.Printf("Created new Black & White tag with ID: %d", id)
	return nil
}

// getPhotosToProcess retrieves photos that need processing
func (app *App) getPhotosToProcess() ([]Photo, error) {
	query := `
		SELECT p.id, p.type, p.checksum,
		       sv_large.short_path as large_path,
		       sv_original.short_path as original_path
		FROM photos p
		LEFT JOIN photos_tags pt ON p.id = pt.photo_id AND pt.tag_id = ?
		LEFT JOIN size_variants sv_large ON p.id = sv_large.photo_id AND sv_large.type = 2
		LEFT JOIN size_variants sv_original ON p.id = sv_original.photo_id AND sv_original.type = 0
		WHERE pt.photo_id IS NULL
		AND p._dz_bw IS NULL
		AND p.type NOT LIKE '%video%'
		AND p.type NOT LIKE '%raw%'
		ORDER BY p.created_at ASC
		LIMIT 100
	`

	rows, err := app.db.Query(query, app.bwTagID)
	if err != nil {
		return nil, fmt.Errorf("failed to query photos: %w", err)
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var photo Photo
		var largePath, originalPath sql.NullString

		if err := rows.Scan(
			&photo.ID, &photo.Type, &photo.Checksum,
			&largePath, &originalPath,
		); err != nil {
			return nil, fmt.Errorf("failed to scan photo: %w", err)
		}

		if largePath.Valid {
			photo.LargePath = largePath.String
		}
		if originalPath.Valid {
			photo.OriginalPath = originalPath.String
		}

		photos = append(photos, photo)
	}

	return photos, nil
}

// Photo represents a photo record from the database
type Photo struct {
	ID            string
	Type          string
	Checksum      string
	LargePath     string
	OriginalPath  string
	IsGrayscale   *bool
	Processed     bool
}

// processPhotos processes all photos that need grayscale detection
func (app *App) processPhotos() error {
	for {
		photos, err := app.getPhotosToProcess()
		if err != nil {
			return fmt.Errorf("failed to get photos to process: %w", err)
		}

		if len(photos) == 0 {
			app.logger.Println("No more photos to process")
			break
		}

		app.logger.Printf("Processing %d photos...", len(photos))

		for _, photo := range photos {
			if err := app.processPhoto(&photo); err != nil {
				app.logger.Printf("Failed to process photo %s: %v", photo.ID, err)
				continue
			}
		}

		// Small delay to prevent overwhelming the server
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// processPhoto handles individual photo processing
func (app *App) processPhoto(photo *Photo) error {
	if app.verbose {
		app.logger.Printf("Processing photo: %s (%s)", photo.ID, photo.Type)
	}

	// Download and analyze image
	img, err := app.downloadAndAnalyzeImage(photo)
	if err != nil {
		return fmt.Errorf("failed to download/analyze image: %w", err)
	}

	// Check if grayscale
	isGrayscale, err := imageanalyzer.IsGrayscale(img, app.config.GrayscaleTolerance)
	if err != nil {
		return fmt.Errorf("failed to analyze grayscale: %w", err)
	}

	// Update database
	if err := app.updatePhotoProcessing(photo.ID, isGrayscale); err != nil {
		return fmt.Errorf("failed to update photo: %w", err)
	}

	if isGrayscale {
		if app.verbose {
			app.logger.Printf("Photo %s is grayscale, applying tag", photo.ID)
		}
		if err := app.applyBWTag(photo.ID); err != nil {
			app.logger.Printf("Failed to apply Black & White tag to photo %s: %v", photo.ID, err)
		}
	} else if app.verbose {
		app.logger.Printf("Photo %s is not grayscale", photo.ID)
	}

	return nil
}

// downloadAndAnalyzeImage downloads image data and decodes it
func (app *App) downloadAndAnalyzeImage(photo *Photo) (image.Image, error) {
	// Try Large first, then fallback to Original
	paths := []string{}
	if photo.LargePath != "" {
		paths = append(paths, photo.LargePath)
	}
	if photo.OriginalPath != "" {
		paths = append(paths, photo.OriginalPath)
	}

	var lastErr error
	for _, path := range paths {
		url := app.config.ImageBaseURL + path
		if app.verbose {
			app.logger.Printf("Downloading: %s", url)
		}

		resp, err := http.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		// Read image data
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		// Try to decode as standard image formats first
		reader := strings.NewReader(string(data))
		img, format, err := image.Decode(reader)
		if err != nil {
			// Try WebP specifically
			reader = strings.NewReader(string(data)) // reset reader
			img, err = webp.Decode(reader)
			if err != nil {
				lastErr = fmt.Errorf("failed to decode image (tried standard formats and WebP): %w", err)
				continue
			}
			format = "webp"
		}

		if app.verbose {
			app.logger.Printf("Successfully decoded image format: %s", format)
		}
		return img, nil
	}

	return nil, fmt.Errorf("failed to download any image variant, last error: %w", lastErr)
}

// updatePhotoProcessing updates the photo record with processing results
func (app *App) updatePhotoProcessing(photoID string, isGrayscale bool) error {
	bwValue := 0
	if isGrayscale {
		bwValue = 1
	}

	_, err := app.db.Exec("UPDATE photos SET _dz_bw = ?, updated_at = NOW() WHERE id = ?",
		bwValue, photoID)
	return err
}

// applyBWTag applies the Black & White tag to a photo
func (app *App) applyBWTag(photoID string) error {
	_, err := app.db.Exec("INSERT IGNORE INTO photos_tags (tag_id, photo_id) VALUES (?, ?)",
		app.bwTagID, photoID)
	return err
}