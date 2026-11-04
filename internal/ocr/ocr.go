package ocr

import (
	"os"
	"os/exec"
)

// Available reports whether tesseract is installed.
func Available() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

// ExtractText runs tesseract on imagePath and returns raw text. Returns "" if
// tesseract is unavailable or fails.
func ExtractText(imagePath string) string {
	if !Available() {
		return ""
	}
	out, err := exec.Command("tesseract", imagePath, "stdout").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// WriteTemp writes data to a temp file and returns its path.
func WriteTemp(data []byte, suffix string) (string, error) {
	f, err := os.CreateTemp("", "receipt-*"+suffix)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return f.Name(), nil
}
