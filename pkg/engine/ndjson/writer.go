// Package ndjson écrit des events NDJSON (Newline Delimited JSON) sur un io.Writer.
// Utilisé pour la sortie streaming du moteur, consommée par la GUI ou par jq.
package ndjson

import (
	"encoding/json"
	"io"
)

// Writer émet des events JSON ligne par ligne.
type Writer struct {
	w io.Writer
}

// NewWriter retourne un Writer qui écrit sur w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Emit sérialise event en JSON compact suivi d'un \n.
// Retourne la première erreur rencontrée (Marshal ou Write).
func (w *Writer) Emit(event any) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err = w.w.Write([]byte{'\n'})
	return err
}
