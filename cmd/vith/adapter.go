package main

import (
	"fmt"

	"github.com/ViBiOh/absto/pkg/absto"
	model "github.com/ViBiOh/absto/pkg/model"
)

type adapters struct {
	storage model.Storage
}

func newAdapters(config configuration, clients clients) (adapters, error) {
	var output adapters
	var err error

	output.storage, err = absto.New(config.absto, clients.telemetry.TracerProvider())
	if err != nil {
		return output, fmt.Errorf("absto: %w", err)
	}

	return output, nil
}
