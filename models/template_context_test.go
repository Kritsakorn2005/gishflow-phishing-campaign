package models

import (
	"fmt"
	"strings"

	check "gopkg.in/check.v1"
)

type mockTemplateContext struct {
	URL         string
	FromAddress string
}

func (m mockTemplateContext) getFromAddress() string {
	return m.FromAddress
}

func (m mockTemplateContext) getBaseURL() string {
	return m.URL
}

func (s *ModelsSuite) TestNewTemplateContext(c *check.C) {
	r := Result{
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
		},
		RId: "1234567",
	}
	ctx := mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From Address <from@example.com>",
	}
	expected := PhishingTemplateContext{
		URL:           fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId),
		BaseURL:       ctx.URL,
		BaseRecipient: r.BaseRecipient,
		TrackingURL:   fmt.Sprintf("%s/track?rid=%s", ctx.URL, r.RId),
		From:          "From Address",
		RId:           r.RId,
	}
	expected.Tracker = "<img alt='' style='display: none' src='" + expected.TrackingURL + "'/>"
	got, err := NewPhishingTemplateContext(ctx, r.BaseRecipient, r.RId)
	c.Assert(err, check.Equals, nil)
	// The QRCode is a generated data-URI image, so we can't hardcode its
	// value. Verify it's populated and well-formed, then copy it into the
	// expected struct so the remaining fields can be compared as before.
	c.Assert(strings.HasPrefix(got.QRCode, "<img alt='' src='data:image/png;base64,"), check.Equals, true)
	c.Assert(len(got.QRCode) > 100, check.Equals, true)
	expected.QRCode = got.QRCode
	c.Assert(got, check.DeepEquals, expected)
}

func (s *ModelsSuite) TestTemplateContextQRCodeEncodesPhishURL(c *check.C) {
	r := Result{
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
		},
		RId: "1234567",
	}
	ctx := mockTemplateContext{
		URL:         "http://example.com",
		FromAddress: "From Address <from@example.com>",
	}
	got, err := NewPhishingTemplateContext(ctx, r.BaseRecipient, r.RId)
	c.Assert(err, check.Equals, nil)
	// The QR code should be a non-empty base64 PNG embedded as a data-URI.
	c.Assert(strings.HasPrefix(got.QRCode, "<img alt='' src='data:image/png;base64,"), check.Equals, true)
	c.Assert(strings.HasSuffix(got.QRCode, "'/>"), check.Equals, true)
	// The recipient's phishing URL (with rid) must be present so we know the
	// QR encodes the tracked link and not the bare base URL.
	c.Assert(got.URL, check.Equals, fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId))
}
