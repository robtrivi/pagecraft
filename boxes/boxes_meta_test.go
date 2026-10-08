package boxes

import (
	"testing"
)

// Metadata step 1: <title> and author/description meta values reach the root.
func TestBoxes_Metadata_FromHead(t *testing.T) {
	root, err := Build(`<html><head><title>My Document</title><meta name="author" content="Anne"><meta name="description" content="Test"></head><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Meta.Title != "My Document" {
		t.Errorf("title: got %q", root.Meta.Title)
	}
	if root.Meta.Author != "Anne" {
		t.Errorf("autor: got %q", root.Meta.Author)
	}
	if root.Meta.Description != "Test" {
		t.Errorf("description: got %q", root.Meta.Description)
	}
}

func TestBoxes_Metadata_Empty(t *testing.T) {
	root, err := Build(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Meta != (Metadata{}) {
		t.Errorf("without head: got %+v", root.Meta)
	}
}
