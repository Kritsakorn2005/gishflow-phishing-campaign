package models

import (
	"bytes"
	"net/mail"
	"net/url"
	"path"
	"text/template"
)

// TemplateContext is an interface that allows both campaigns and email
// requests to have a PhishingTemplateContext generated for them.
type TemplateContext interface {
	getFromAddress() string
	getBaseURL() string
}

// PhishingTemplateContext is the context that is sent to any template, such
// as the email or landing page content.
type PhishingTemplateContext struct {
	From        string
	URL         string
	Tracker     string
	TrackingURL string
	RId         string
	BaseURL     string
	QRCode      string
	BaseRecipient
}

// NewPhishingTemplateContext returns a populated PhishingTemplateContext,
// parsing the correct fields from the provided TemplateContext and recipient.
func NewPhishingTemplateContext(ctx TemplateContext, r BaseRecipient, rid string) (PhishingTemplateContext, error) {
	// Parse the "From" for the {{.From}} variable. A LINE (or other non-email)
	// channel has no email From address, so we tolerate an empty/invalid value
	// rather than failing - email campaigns always have a valid From.
	fn := ""
	if f, ferr := mail.ParseAddress(ctx.getFromAddress()); ferr == nil {
		fn = f.Name
		if fn == "" {
			fn = f.Address
		}
	}
	templateURL, err := ExecuteTemplate(ctx.getBaseURL(), r)
	if err != nil {
		return PhishingTemplateContext{}, err
	}

	// For the base URL, we'll reset the the path and the query
	// This will create a URL in the form of http://example.com
	baseURL, err := url.Parse(templateURL)
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	baseURL.Path = ""
	baseURL.RawQuery = ""

	phishURL, _ := url.Parse(templateURL)
	q := phishURL.Query()
	q.Set(RecipientParameter, rid)
	phishURL.RawQuery = q.Encode()

	trackingURL, _ := url.Parse(templateURL)
	trackingURL.Path = path.Join(trackingURL.Path, "/track")
	trackingURL.RawQuery = q.Encode()

	// Generate a QR code that encodes the recipient's phishing URL, tagged with
	// a "qr" marker so scans can be distinguished from ordinary link clicks. It's
	// embedded as a self-contained data-URI <img> so templates can simply
	// reference {{.QRCode}}. Any generation error leaves QRCode empty so a
	// campaign that doesn't use the QR code is never affected.
	qrURL, _ := url.Parse(phishURL.String())
	qq := qrURL.Query()
	qq.Set("qr", "1")
	qrURL.RawQuery = qq.Encode()
	// GishFlow: reference the QR as a hosted image on the phishing server
	// (baseURL/qr?u=...) instead of an inline data: URI, because Gmail (and some
	// other clients) strip data: images. Fetching the image only renders the
	// code; the scan is tracked when the encoded URL is actually visited.
	qrImg, _ := url.Parse(baseURL.String())
	qrImg.Path = path.Join(qrImg.Path, "/qr")
	qrImg.RawQuery = url.Values{"u": {qrURL.String()}}.Encode()
	qrCode := "<img alt='' src='" + qrImg.String() + "'/>"

	return PhishingTemplateContext{
		BaseRecipient: r,
		BaseURL:       baseURL.String(),
		URL:           phishURL.String(),
		TrackingURL:   trackingURL.String(),
		Tracker:       "<img alt='' style='display: none' src='" + trackingURL.String() + "'/>",
		From:          fn,
		RId:           rid,
		QRCode:        qrCode,
	}, nil
}

// ExecuteTemplate creates a templated string based on the provided
// template body and data.
func ExecuteTemplate(text string, data interface{}) (string, error) {
	buff := bytes.Buffer{}
	tmpl, err := template.New("template").Parse(text)
	if err != nil {
		return buff.String(), err
	}
	err = tmpl.Execute(&buff, data)
	return buff.String(), err
}

// ValidationContext is used for validating templates and pages
type ValidationContext struct {
	FromAddress string
	BaseURL     string
}

func (vc ValidationContext) getFromAddress() string {
	return vc.FromAddress
}

func (vc ValidationContext) getBaseURL() string {
	return vc.BaseURL
}

// ValidateTemplate ensures that the provided text in the page or template
// uses the supported template variables correctly.
func ValidateTemplate(text string) error {
	vc := ValidationContext{
		FromAddress: "foo@bar.com",
		BaseURL:     "http://example.com",
	}
	td := Result{
		BaseRecipient: BaseRecipient{
			Email:     "foo@bar.com",
			FirstName: "Foo",
			LastName:  "Bar",
			Position:  "Test",
		},
		RId: "123456",
	}
	ptx, err := NewPhishingTemplateContext(vc, td.BaseRecipient, td.RId)
	if err != nil {
		return err
	}
	_, err = ExecuteTemplate(text, ptx)
	if err != nil {
		return err
	}
	return nil
}
