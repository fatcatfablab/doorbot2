package sender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fatcatfablab/doorbot2/db"
)

const (
	doordDateFmt = "01/02/2006"
	doordTimeFmt = time.TimeOnly
)

type DoordSender struct {
	client *http.Client
	url    *url.URL
}

func NewDoord(url *url.URL) *DoordSender {
	return newWithHttpClient(url, &http.Client{})
}

func newWithHttpClient(url *url.URL, c *http.Client) *DoordSender {
	return &DoordSender{url: url, client: c}
}

func (d *DoordSender) Post(ctx context.Context, stats db.Stats) error {
	var b bytes.Buffer
	fmt.Fprintf(
		&b,
		"%s,%s,%s,%d",
		stats.Last.Format(doordDateFmt),
		stats.Last.Format(doordTimeFmt),
		stats.Name,
		1,
	)
	req := &http.Request{
		Method: http.MethodPost,
		URL:    d.url,
		Body:   io.NopCloser(&b),
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending doord request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("doord request returned: %s", resp.Status)
	}
	defer resp.Body.Close()

	return nil
}
