# Lychee Black & White Tagger

A Go application that automatically detects grayscale photos in a Lychee installation and tags them as "Black & White".

## Features

- Connects to Lychee MySQL database
- Adds `_dz_bw` column to photos table (if not exists)
- Downloads and analyzes photos for grayscale content
- Tags grayscale photos with "Black & White" tag
- Skips video files and RAW formats
- Configurable grayscale tolerance
- Resilient error handling with logging
- Supports JPEG, PNG, and WebP formats

## Configuration

Copy `config.yml.example` to `config.yml` and modify the settings:

```yaml
database:
  host: localhost
  port: 3306
  username: lychee_user
  password: your_password_here
  database: lychee

grayscale_tolerance: 0.1

image_base_url: "https://your-lychee-instance.com/public/"
```

### Configuration Options

- **database**: MySQL connection settings
  - `host`: Database server hostname
  - `port`: Database server port (default: 3306)
  - `username`: Database username
  - `password`: Database password
  - `database`: Lychee database name
- **grayscale_tolerance**: Tolerance for grayscale detection (0.0 to 1.0)
  - 0.0 = Strict grayscale detection
  - 1.0 = Very permissive
  - 0.1 = Recommended starting value
- **image_base_url**: Base URL where Lychee stores image files

## Usage

### Basic Usage

```bash
./lychee-bw-tagger
```

### With Custom Config

```bash
./lychee-bw-tagger -config /path/to/config.yml
```

### Verbose Logging

```bash
./lychee-bw-tagger -verbose
```

## Building

```bash
go build -o lychee-bw-tagger
```

## How It Works

1. **Database Setup**: Adds `_dz_bw` column to photos table if it doesn't exist
2. **Tag Management**: Finds or creates "Black & White" tag
3. **Photo Selection**: Gets photos that:
   - Don't have the Black & White tag
   - Have `_dz_bw` set to NULL
   - Are not video or RAW files
4. **Processing Loop**: For each photo:
   - Downloads Large variant (falls back to Original)
   - Analyzes for grayscale content
   - Updates `_dz_bw` column (true/false)
   - Applies Black & White tag if grayscale
5. **Resilience**: Continues processing even if individual photos fail

## Database Schema Changes

The application adds one column to the `photos` table:

```sql
ALTER TABLE photos
ADD COLUMN _dz_bw TINYINT(1) NULL COMMENT 'Black & white detection result';
```

## Image Format Support

- JPEG (.jpg, .jpeg)
- PNG (.png)
- WebP (.webp)

Video files and RAW formats are automatically skipped.

## Logging

The application uses the standard `log` package for output:

- Standard messages show general progress
- Use `-verbose` flag for detailed per-photo information
- Errors are logged with context for debugging

## Error Handling

- Network failures: Retries with fallback image variants
- Database errors: Logged and continue processing other photos
- Image decode errors: Skips problematic files
- Schema issues: Automatically adds missing columns

## Dependencies

- `github.com/go-sql-driver/mysql` - MySQL driver
- `gopkg.in/yaml.v3` - YAML configuration parsing
- `github.com/cdzombak/image-analyzer-go` - Grayscale detection
- `golang.org/x/image/webp` - WebP format support

## License

MIT