package transfer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

type Progress func(downloaded, total int64)

func Download(ctx context.Context, client *http.Client, urls []string, destination string, resume bool, progress Progress) error {
	if len(urls) == 0 {
		return fmt.Errorf("download has no source URLs")
	}
	part := destination + ".part"
	var lastErr error
	for _, source := range urls {
		if err := downloadOne(ctx, client, source, part, resume, progress); err != nil {
			lastErr = err
			continue
		}
		if err := os.Rename(part, destination); err != nil {
			return err
		}
		return nil
	}
	return lastErr
}

func downloadOne(ctx context.Context, client *http.Client, source, part string, resume bool, progress Progress) error {
	var offset int64
	if resume {
		if info, err := os.Stat(part); err == nil {
			offset = info.Size()
		}
	} else if err := os.Remove(part); err != nil && !os.IsNotExist(err) {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0")
	request.Header.Set("Referer", "https://www.bilibili.com")
	if offset > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if offset > 0 && response.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		response.Body.Close()
		if err := os.Remove(part); err != nil && !os.IsNotExist(err) {
			return err
		}
		return downloadOne(ctx, client, source, part, false, progress)
	}
	if response.StatusCode == http.StatusPartialContent {
		var start, end, size int64
		if _, err := fmt.Sscanf(response.Header.Get("Content-Range"), "bytes %d-%d/%d", &start, &end, &size); err != nil || start != offset || end < start || size <= end {
			return fmt.Errorf("GET %s: invalid Content-Range %q for offset %d", source, response.Header.Get("Content-Range"), offset)
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 && response.StatusCode == http.StatusPartialContent {
		flags |= os.O_APPEND
	} else {
		offset = 0
		flags |= os.O_TRUNC
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("GET %s: %s", source, response.Status)
	}
	file, err := os.OpenFile(part, flags, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	total := response.ContentLength
	if total >= 0 {
		total += offset
	}
	written := offset
	buffer := make([]byte, 128*1024)
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			if _, err := file.Write(buffer[:n]); err != nil {
				return err
			}
			written += int64(n)
			if progress != nil {
				progress(written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if total >= 0 && written != total {
		return fmt.Errorf("GET %s: downloaded %d bytes, expected %d", source, written, total)
	}
	return file.Sync()
}
