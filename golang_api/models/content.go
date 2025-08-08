package models

import (
	"fmt"
	"errors"
	"time"
	"gorm.io/gorm"
)

type Content struct {
	gorm.Model
	Title               string    `json:"title" gorm:"default:'unknown'"`
	Artist              string    `json:"artist" gorm:"default:'unknonw'"`
	Url                 string    `json:"url"`
	SourceUrl           string    `json:"source_url"`
	PlayCount           string    `json:"play_count" gorm:"default:'1'"`
	Energy              string    `json:"energy"`
	NormEnergy          string    `json:"norm_energy"`
	Key                 string    `json:"key"`
	Centroid            string    `json:"centroid"`
	NormCentroid        string    `json:"norm_centroid"`
	ReplaygainTrackGain string    `json:"replaygain_track_gain"`
	ReplaygainTrackPeak string    `json:"replaygain_track_peak"`
	Offset              string    `json:"liq_on_offset" gorm:"column:liq_on_offset"`
	Stream              string    `json:"stream"`
	Stream_2            string    `json:"stream_2"`
	MixType             string    `json:"mix_type"`
	Style               string    `json:"style"`
	LastPlayed          time.Time `json:"last_played"`
	SpecFlatnessNorm    string    `json:"spec_flatness_norm"`
	YamnetEmbeddings    []byte    `json:"yamnet_embedding" grom:"type:blob"`
	PCA1D               float64   `json:"pca_1d"`
	IntensityStep       int       `json:"intersity_step"`
	Duration            string    `json:"duration"`
	Processed           string    `json:"processed"`
	Currated            string    `json:"currated"`
}

// Validate Content item
func (c Content) Validate() error {
	fmt.Println("validating")
	// Add your validation rules here
	if c.Title == "" {
		return errors.New("Title is required")
	}
	if c.Artist == "" {
		return errors.New("Artist is required")
	}
	if c.Url == "" {
		return errors.New("Url is required")
	}
	return nil
}
