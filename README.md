# File Conversion Utilities

This directory contains utilities for converting various file formats to supported formats.

## .doc to docx Conversion

The `.doc` file reader converts Microsoft Word documents (`.doc` format) to docx using LibreOffice, then processes them through the existing docx workflow.

### Prerequisites

To use `.doc` file support, you need to install LibreOffice on your system.

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install libreoffice
```

#### CentOS/RHEL/Fedora
```bash
# CentOS/RHEL
sudo yum install libreoffice

# Fedora
sudo dnf install libreoffice
```

#### macOS
```bash
# Using Homebrew
brew install --cask libreoffice

# Or download from https://www.libreoffice.org/download/download/
```

### Verification

After installation, verify LibreOffice is available:
```bash
libreoffice --version
# or
soffice --version
```

### How It Works

1. When a `.doc` file is uploaded, the system detects the file extension
2. The file is temporarily saved to disk
3. LibreOffice converts the `.doc` file to docx using the command:
   ```bash
   libreoffice --headless --convert-to docx --outdir <output_dir> <input_file>
   ```
4. The converted docx is processed using the existing docx reader
5. Temporary files are cleaned up
6. The extracted text is returned to the user

### Error Handling

If LibreOffice is not installed or fails to convert:
- The system will return an error message
- Users will be prompted to install LibreOffice or convert the file manually
- The original file upload will fail gracefully

### Performance Notes

- Conversion time depends on file size and complexity
- Large `.doc` files may take longer to process
- The conversion process is synchronous and may block other operations
- Consider implementing async processing for production use with large files

### Security Considerations

- Temporary files are created in a controlled directory
- Files are cleaned up after processing
- The conversion process runs in the same process context
- Consider sandboxing for production environments 

## Custom DOC->DOCX External Service (customDoc2Docx)

This project can use a custom external service to convert `.doc` to `.docx` before falling back to local LibreOffice. Configure it via `systemEnv.customDoc2Docx`.

### Configuration

- **Location**: `systemEnv.customDoc2Docx` (e.g., `projects/app/data/config.json`)
- **Fields**:
  - `url` (string): Endpoint that accepts the `.doc` and returns a `.docx` (required to enable external conversion)
  - `key` (string): Optional bearer token used as `Authorization: Bearer <key>`
  - `timeoutMs` (number): Request timeout in milliseconds (default 600000)

Example (`projects/app/data/config.json`):

```json
{
  "systemEnv": {
    "customDoc2Docx": {
      "url": "https://your-docx.example.com/convert/doc2docx",
      "key": "YOUR_SECRET_KEY",
      "timeoutMs": 600000
    }
  }
}
```

### Request

- **Method**: `POST`
- **URL**: `systemEnv.customDoc2Docx.url`
- **Headers**:
  - `Authorization: Bearer <key>` when `systemEnv.customDoc2Docx.key` is set
  - `Content-Type`: `multipart/form-data` (boundary handled automatically)
- **Body**: multipart form with a single file field
  - Field name: `file`
  - Value: the original `.doc` file bytes; filename preserved

Curl example:

```bash
curl -X POST "https://your-docx.example.com/convert" \
  -H "Authorization: Bearer YOUR_SECRET_KEY" \
  -F "file=@/path/to/input.doc" \
  --output output.docx
```

### Response

Your service may respond in one of two supported formats:

1) **Binary DOCX**
- **Status**: 200
- **Content-Type**: `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- **Body**: raw `.docx` bytes

2) **JSON containing Base64 DOCX**
- **Status**: 200
- **Content-Type**: `application/json`
- **Body**: an object carrying the base64-encoded `.docx` in one of these fields (first found is used):
  - `docxBase64` (preferred)
  - `base64`
  - `data`

Example JSON response:

```json
{
  "docxBase64": "UEsDBBQAAAAIA...base64...AAA="
}
```

Notes:
- If `Content-Type` is unknown, the client treats it as binary and writes bytes to a `.docx` file.
- Output filename is derived from the original `.doc` name with `.docx` extension.

### Error handling and fallback

- If `customDoc2Docx.url` is not set, the external call is skipped.
- If the external service fails, times out, or returns unexpected JSON, the system logs a warning and falls back to local LibreOffice conversion (if available).
- Typical errors include:
  - Network/timeout
  - Non-200 responses
  - JSON without a valid `docxBase64`/`base64`/`data` string

### Minimal service reference implementation (pseudo)

```http
POST /convert/doc2docx
Authorization: Bearer <optional>
Content-Type: multipart/form-data; boundary=...

form-data; name="file"; filename="input.doc"
<.doc bytes>
---

Response (option A):
200 OK
Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document
<.docx bytes>

Response (option B):
200 OK
Content-Type: application/json
{"docxBase64": "<base64 of .docx>"}
```
