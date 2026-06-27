# Generated code is checked in; run `make generate` after editing proto/ or internal/model.

GORM_MODEL_DIR := ./internal/model
GORM_OUT_DIR   := ./internal/generated

.PHONY: generate generate-proto generate-model

## generate: regenerate both protobuf stubs and GORM query helpers
generate: generate-proto generate-model

## generate-proto: regenerate Go + web stubs from proto/ via buf
generate-proto:
	buf generate

## generate-model: regenerate GORM field helpers from the model structs
generate-model:
	go tool gorm gen -i $(GORM_MODEL_DIR) -o $(GORM_OUT_DIR)
