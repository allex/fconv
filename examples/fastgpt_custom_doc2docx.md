## Custom DOC->DOCX External Service (customDoc2Docx)

This guide explains how to integrate an external DOC to DOCX conversion service with FastGPT to enhance document processing capabilities. This feature is particularly useful in scenarios where:

- You need to process legacy `.doc` files in your FastGPT datasets
- You want to use a specialized conversion service with custom formatting rules
- You prefer using a cloud-based conversion service over local LibreOffice
- You need to handle high-volume document conversions with better scalability

The service acts as a pre-processor that converts `.doc` files to `.docx` format before they are processed by FastGPT's document analysis pipeline. If the external service is unavailable, the system automatically falls back to local LibreOffice conversion.

Use the `fconv` service as a custom external service to configure this functionality via `systemEnv.customDoc2Docx`.

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
      "url": "https://your-docx.example.com/api/v1/convert/doc2docx",
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
curl -X POST "https://your-docx.example.com/api/v1/convert/doc2docx" \
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
  "base64": "UEsDBBQAAAAIA...base64...AAA="
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
POST /api/v1/convert/doc2docx
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
