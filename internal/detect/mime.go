package detect

import (
	"io"
	"net/http"
	"os"
	"strings"
)

// nonRegular reports whether path is something we must never read from.
// FIFOs, devices, and sockets open successfully but block on read; directories
// open but fail with EISDIR. Symlinks are followed so a link to a regular file
// still gets sniffed. A path we could not stat at all is left to the caller's
// Open, which will produce a real error.
func nonRegular(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.Mode().IsRegular()
}

func MIME(path string) string {
	if nonRegular(path) {
		return "application/octet-stream"
	}

	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF && n == 0 {
		// A file we could not read from at all is opaque. An empty file is
		// text: http.DetectContentType returns text/plain for zero bytes.
		return "application/octet-stream"
	}

	return http.DetectContentType(buf[:n])
}

func IsBinary(mime string) bool {
	if strings.HasPrefix(mime, "text/") {
		return false
	}

	switch mime {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/x-sh",
		"application/x-shellscript",
		"application/x-yaml",
		"application/toml",
		"application/x-httpd-php",
		"application/rtf",
		"application/xhtml+xml",
		"application/sql",
		"application/x-ndjson",
		"application/ld+json",
		"application/problem+json",
		"application/problem+xml",
		"application/geo+json",
		"application/manifest+json":
		return false
	}

	if strings.HasPrefix(mime, "image/") ||
		strings.HasPrefix(mime, "audio/") ||
		strings.HasPrefix(mime, "video/") {
		return true
	}

	switch mime {
	case "application/octet-stream",
		"application/zip",
		"application/gzip",
		"application/x-tar",
		"application/x-bzip",
		"application/x-bzip2",
		"application/x-7z-compressed",
		"application/x-rar-compressed",
		"application/x-xz",
		"application/x-lzip",
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/x-executable",
		"application/x-sharedlib",
		"application/x-mach-binary",
		"application/x-dosexec",
		"application/wasm",
		"application/java-archive",
		"application/x-java-archive",
		"application/x-elf",
		"application/x-pie-executable",
		"application/x-object",
		"application/x-archive",
		"application/font-sfnt",
		"application/font-woff",
		"application/x-font-ttf",
		"application/x-font-otf",
		"font/ttf",
		"font/otf",
		"font/woff",
		"font/woff2",
		"application/x-protobuf",
		"application/x-thrift",
		"application/x-msgpack",
		"application/cbor",
		"application/x-pkcs12",
		"application/x-x509-ca-cert",
		"application/pkix-cert",
		"application/pgp-encrypted",
		"application/pgp-signature",
		"application/ogg",
		"application/x-shockwave-flash":
		return true
	}

	return true
}

func DetectFromFile(path string) (mime string, isBinary bool, err error) {
	if nonRegular(path) {
		return "application/octet-stream", true, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream", true, err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, readErr := f.Read(buf)
	if readErr != nil && n == 0 {
		m := MIME(path)
		return m, IsBinary(m), nil
	}

	mime = http.DetectContentType(buf[:n])
	return mime, IsBinary(mime), nil
}
