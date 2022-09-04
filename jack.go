package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Validador de UF
var UFValidator map[string]string

// Definição do schema de UF
type FederativeUnit struct {
	Name       string     `json:"uf" bson:"uf"`
	Localities []Locality `json:"localidades" bson:"localidades"`
}

// Definição do schema da localidade
type Locality struct {
	Id       string `json:"id" bson:"id"`
	Name     string `json:"localidade" bson:"localidade"`
	CEPRange string `json:"faixa de cep" bson:"faixa de cep"`
}

func main() {
	router := localitiesHandler()
	s := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  600 * time.Second,
		WriteTimeout: 600 * time.Second,
	}
	s.ListenAndServe()
	router.Run()
}

// Handler das Rotas
func localitiesHandler() *gin.Engine {
	router := gin.Default()
	v1 := router.Group("/v1")
	v1.GET("/localidades/:ufs", getLocalities)
	return router
}

// Controlador da localidades
func getLocalities(c *gin.Context) {
	params := c.Param("ufs")
	options := strings.Split(params, ",")
	if len(options) > 5 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	resultCh := make(chan []Locality)

	var answers []FederativeUnit
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	for _, value := range options {
		fmt.Println("Crawler execution for UF: " + value)
		go func() {
			value = strings.Trim(value, " ")
			isValid, err := UFIsValid(value)
			if !isValid {
				c.AbortWithError(http.StatusNotFound, err)
				close(resultCh)
				return
			}
			err = crawlerExecution(ctx, value, resultCh)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}()
		localities := <-resultCh
		finalResult := FederativeUnit{
			Name:       value,
			Localities: localities,
		}
		answers = append(answers, finalResult)
	}
	jsonlResponse(answers)
	c.AbortWithStatus(http.StatusOK)
}

// Monta resultado da pesquisa em arquivo
func jsonlResponse(answers []FederativeUnit) error {
	file, err := os.Create("result.jsonl")
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range answers {
		bytes, err := json.Marshal(line)
		if err != nil {
			log.Fatal(err)
		}

		if len(line.Localities) > 0 {
			bytes_to_string := string(bytes)
			fmt.Fprintln(w, bytes_to_string)
		}

	}
	return w.Flush()
}

// Função principal de crawling para busca de localidades por paginação via chromedp
func crawlerExecution(ctx context.Context, uf string, op chan []Locality) error {
	fmt.Println("Crawler execution for UF: " + uf)
	UF := uf
	var domHTML string
	Results := []Locality{}

	chromeDpTask := chromedp.Tasks{
		chromedp.Navigate("https://www2.correios.com.br/sistemas/buscacep/buscaFaixaCep.cfm"),
		chromedp.WaitVisible("#Geral select"),
		chromedp.SetAttributeValue(`#Geral select option[value="`+UF+`"]`, "selected", "true"),
		chromedp.WaitSelected(`#Geral select option[value="` + UF + `"]`),
		chromedp.Click(`#Geral input[value="Buscar"]`),
		chromedp.Sleep(1 * time.Second),
		chromedp.OuterHTML(`div[class*="ctrlcontent"]`, &domHTML),
	}

	err := chromedp.Run(ctx, chromeDpTask)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(domHTML))
	if err != nil {
		return err
	}

	Results, err = extractTableData(doc, Results, false)
	if err != nil {
		return err
	}

	haveNextButton := doc.Find(`form[name="Proxima"]`)
	for haveNextButton.Size() > 0 {
		//Encontrei nova página
		doc, err = whileHaveNext(ctx)
		if err != nil {
			return err
		}
		haveNextButton = doc.Find(`form[name="Proxima"]`)

		Results, err = extractTableData(doc, Results, true)
		if err != nil {
			return err
		}
	}
	time.Sleep(time.Millisecond * 10)
	op <- Results
	return nil
}

// Extrai dados da tabela de localidades
func extractTableData(doc *goquery.Document, Results []Locality, isNextButton bool) ([]Locality, error) {
	tableIndex := "1"
	if !isNextButton {
		tableIndex = "2"
	}
	if doc.Find(`table[class*="tmptabela"]:nth-of-type(`+tableIndex+`)`).Size() > 0 {
		doc.Find(`table[class*="tmptabela"]:nth-of-type(` + tableIndex + `) tbody tr`).Each(func(i int, selection *goquery.Selection) {
			// Monta objeto de localidade
			uuid := uuid.New()
			locality := Locality{}
			locality.Id = uuid.String()
			locality.Name = selection.Find("td:nth-child(1)").Text()
			locality.CEPRange = selection.Find("td:nth-child(2)").Text()
			//ignora quando campo vir vazio
			if locality.Name != "" {
				Results = append(Results, locality)
			}
		})
	}
	return Results, nil
}

// Enquanto houver botão de próxima página, executa ação de click
func whileHaveNext(ctx context.Context) (*goquery.Document, error) {
	var domHTML string
	chromeDpTask2 := chromedp.Tasks{
		chromedp.WaitVisible(`form[name="Proxima"]`),
		chromedp.Click(`div[class*="ctrlcontent"] div[style="float:left"]:nth-of-type(2)`),
		chromedp.Sleep(1 * time.Second),
		chromedp.WaitVisible(`table[class*="tmptabela"]`),
		chromedp.OuterHTML(`div[class*="ctrlcontent"]`, &domHTML),
	}

	err := chromedp.Run(ctx, chromeDpTask2)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(domHTML))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func UFIsValid(UF string) (bool, error) {
	UFValidator := map[string]string{
		"AC": "AC",
		"AL": "AL",
		"AP": "AP",
		"AM": "AM",
		"BA": "BA",
		"CE": "CE",
		"DF": "DF",
		"ES": "ES",
		"GO": "GO",
		"MA": "MA",
		"MT": "MT",
		"MS": "MS",
		"MG": "MG",
		"PA": "PA",
		"PB": "PB",
		"PR": "PR",
		"PE": "PE",
		"PI": "PI",
		"RJ": "RJ",
		"RN": "RN",
		"RS": "RS",
		"RO": "RO",
		"RR": "RR",
		"SC": "SC",
		"SP": "SP",
		"SE": "SE",
		"TO": "TO",
	}
	_, ok := UFValidator[UF]
	if !ok {
		return false, fmt.Errorf("UF invalida")
	}
	return true, nil
}
