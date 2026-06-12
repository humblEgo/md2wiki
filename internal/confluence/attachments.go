package confluence

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// findAttachmentID looks up the ID of an existing attachment on the page by filename.
// If no attachment with that filename exists, it returns ("", nil) so callers can
// distinguish "not found" from an actual error.
func (c *Client) findAttachmentID(ctx context.Context, pageID, filename string) (string, error) {
	u := c.v1("/content/"+pageID+"/child/attachment") + "?" +
		url.Values{"filename": {filename}}.Encode()
	var resp struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if _, err := c.doJSON(ctx, http.MethodGet, u, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.Results) == 0 {
		return "", nil
	}
	return resp.Results[0].ID, nil
}

// UploadAttachment upserts a file onto a page: if an attachment with the same filename
// already exists it updates the data as a new version, otherwise it creates a new one
// (using the Confluence v1 multipart endpoint). This upsert behavior is what keeps the
// mirror idempotent: re-applying the same docs would otherwise fail with a duplicate
// filename conflict (HTTP 400) on the create endpoint.
func (c *Client) UploadAttachment(ctx context.Context, pageID, filename string, data []byte) error {
	attID, err := c.findAttachmentID(ctx, pageID, filename)
	if err != nil {
		return err
	}
	path := "/content/" + pageID + "/child/attachment"
	if attID != "" {
		path += "/" + attID + "/data" // update the existing attachment's data (creates a new version)
	}

	body, contentType, err := multipartFile(filename, data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.v1(path), body)
	if err != nil {
		return fmt.Errorf("confluence: new request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Atlassian-Token", "no-check")
	_, _, err = c.do(req)
	return err
}

// multipartFile builds a multipart/form-data body containing a single "file" part and
// returns it together with the Content-Type header value (which includes the generated
// boundary). The caller must set that Content-Type on the request for the server to parse it.
func multipartFile(filename string, data []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", fmt.Errorf("confluence: multipart: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return nil, "", fmt.Errorf("confluence: write attachment: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, "", fmt.Errorf("confluence: close multipart: %w", err)
	}
	return &buf, mw.FormDataContentType(), nil
}
