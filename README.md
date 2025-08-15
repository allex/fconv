# File Conversion Utilities (FCONV)

High-performance HTTP server for document format conversions, specializing in .doc to .docx conversion via LibreOffice. Features REST API, pluggable converter architecture, and optional external service integration. [source](https://github.com/allex/fconv)

## Quick Start / Usage

- **Run locally (Go)**:
  - Build and run: `go run .`
  - Or build binary: `go build -o fconv . && ./fconv`
  - Options: `./fconv -h` (flags include `-host`, `-port`; envs include `FCONV_PORT`, `FCONV_LISTEN_ADDR`, `FCONV_AUTH_KEY`, `FCONV_TIMEOUT` etc,.)

- **Run with Docker**:
  - Pull/build image (example tag): `docker pull tdio/fconv:latest`
  - Start server: `docker run --rm -p 8080:8080 --name fconv tdio/fconv:latest`
  - For more help: `docker run --rm tdio/fconv:latest -h`

- **Health check**:
  - `curl http://localhost:8080/healthz` → `ok`

- **Convert .doc → .docx**:
  - Binary response (writes `output.docx`):
    ```bash
    curl -sS -X POST 'http://localhost:8080/api/v1/convert/doc2docx' \
      -F 'file=@/path/to/input.doc' \
      -o output.docx
    ```
  - JSON base64 response:
    ```bash
    curl -sS -H 'Accept: application/json' -X POST 'http://localhost:8080/api/v1/convert/doc2docx' \
      -F 'file=@/path/to/input.doc' | jq -r .base64 | base64 --decode > output.docx
    ```

- **Optional auth**:
  - Start server with `FCONV_AUTH_KEY=secret` and pass header `Authorization: Bearer secret` in requests.

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

## Examples

- [Use fconv as a external doc2docx service](./examples/fastgpt_custom_doc2docx.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
