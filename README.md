# Weather Service with OpenTelemetry and Zipkin

Este projeto implementa dois serviços em Go que utilizam OpenTelemetry (OTEL) e Zipkin para rastreamento distribuído. O Serviço A recebe um CEP via POST e encaminha a solicitação para o Serviço B, que busca a localização e a temperatura atual baseada no CEP fornecido.

## Funcionalidades

### Serviço A
- Recebe um input de 8 dígitos via POST, através do schema: `{ "cep": "29902555" }`
- Valida se o input é válido (contém 8 dígitos) e é uma STRING
- Encaminha a solicitação para o Serviço B via HTTP
- Responde com:
  - HTTP 422 se o CEP for inválido
  - HTTP 200 se o CEP for válido

### Serviço B
- Recebe um CEP válido de 8 dígitos
- Pesquisa o CEP e encontra o nome da localização
- Retorna as temperaturas em Celsius, Fahrenheit e Kelvin juntamente com o nome da localização
- Responde com:
  - HTTP 200 em caso de sucesso
  - HTTP 422 se o CEP for inválido
  - HTTP 404 se o CEP não for encontrado

### Implementação de OTEL e Zipkin
- Implementa tracing distribuído entre Serviço A e Serviço B
- Utiliza spans para medir o tempo de resposta do serviço de busca de CEP e busca de temperatura

## Requisitos
- Go 1.21+
- Docker
- Conta no Google Cloud para configurar o Zipkin se necessário

## Uso

### Executar Localmente

```sh
git clone https://github.com/WellingtonDevBR/tracing-service-go
cd tracing-service-go
WEATHER_API_KEY=98a42a15c266432a98b25526240106
ZIPKIN_ENDPOINT=http://localhost:9411
docker run -d -p 9411:9411 openzipkin/zipkin

cd service-a
go run main.go

cd ../service-b
go run main.go

docker-compose up --build
curl -X POST -H "Content-Type: application/json" -d '{"cep": "05144085"}' http://localhost:8081/cep
```