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
	// The QRCode references a hosted QR image on the phishing server, so we
	// can't hardcode its exact value. Verify it's populated and well-formed,
	// then copy it into the expected struct so the remaining fields compare.
	c.Assert(strings.HasPrefix(got.QRCode, "<img alt='' src='"+ctx.URL+"/qr?u="), check.Equals, true)
	c.Assert(strings.HasSuffix(got.QRCode, "'/>"), check.Equals, true)
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
	// The QR code should be a hosted image on the phishing server.
	c.Assert(strings.HasPrefix(got.QRCode, "<img alt='' src='"+ctx.URL+"/qr?u="), check.Equals, true)
	c.Assert(strings.HasSuffix(got.QRCode, "'/>"), check.Equals, true)
	// The encoded target must carry the recipient's rid and the qr marker, so
	// the QR encodes the tracked link (with scan marker), not the bare base URL.
	c.Assert(strings.Contains(got.QRCode, "rid%3D"+r.RId), check.Equals, true)
	c.Assert(strings.Contains(got.QRCode, "qr%3D1"), check.Equals, true)
	c.Assert(got.URL, check.Equals, fmt.Sprintf("%s?rid=%s", ctx.URL, r.RId))
}
