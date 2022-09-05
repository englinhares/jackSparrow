package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalities(t *testing.T) {
	router := localitiesHandler()

	server := httptest.NewServer(router)
	defer server.Close()

	//UF inexistente
	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/localidades/JI", nil)
	assert.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	//Lista de UFs com espaços em branco
	req, err = http.NewRequest(http.MethodGet, server.URL+"/v1/localidades/PB , CE,AL ", nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)

	//Requisição GET com mais de 5 UFs

	req, err = http.NewRequest(http.MethodGet, server.URL+"/v1/localidades/PB,CE,SP,MT,RJ,PE", nil)
	assert.NoError(t, err)

	res, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}
