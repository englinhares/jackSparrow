# jackSparrow
Web Crawler para busca de informações de localidades e faixas de CEP dos correios

# Install
## Baixar o código do repositório
```
git clone https://github.com/englinhares/jackSparrow.git
```
## Verificar dependências
```
cd jackSparrow
```
```
go mod tidy
```

# Run
Executar o código.
```
go run .
```
O webservice vai subir na porta 9876. 
Agora basta acessar a URL conforme descrito abaixo indicando no último parametro do path a lista de UFs você deseja a busca de dados. Esta lista deve estar separada por vírgula. Ex: SC,SP,PR. O número máximo de UFs esta pré definido em 5.
Via terminal, executar o GET da URL com curl
```
curl http://localhost:9876/v1/localidades/SC
```
Via Browser, abrir o link
* http://localhost:9876/v1/localidades/SC,SP
# Tests
```
go test ./...
```





