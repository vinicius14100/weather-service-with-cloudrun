# API de Serviço Meteorológico

Um serviço em Go que fornece informações meteorológicas com base no CEP brasileiro. O serviço utiliza ViaCEP para obter informações de localização e WeatherAPI para obter dados de temperatura atual.

## 🌐 URL em Produção

**Aplicação implantada no Google Cloud Run:**
```
https://weather-service-679467670189.us-central1.run.app/weather?cep=01153000
```

## 🚀 Funcionalidades

- **Validação de CEP**: Valida CEPs brasileiros de 8 dígitos
- **Consulta de Localização**: Usa API ViaCEP para encontrar cidade pelo CEP
- **Dados Meteorológicos**: Busca temperatura atual da WeatherAPI
- **Conversão de Temperatura**: Retorna temperaturas em Celsius, Fahrenheit e Kelvin
- **Tratamento de Erros**: Códigos HTTP adequados para diferentes cenários de erro

## 📋 Endpoints da API

### GET /weather

Retorna informações meteorológicas atuais para um CEP fornecido.

**Parâmetros de Query:**
- `cep` (string): CEP brasileiro de 8 dígitos

**Resposta de Sucesso (200 OK):**
```json
{
  "temp_C": 28.5,
  "temp_F": 83.3,
  "temp_K": 301.65
}
```

**Respostas de Erro:**
- `422 Unprocessable Entity`: Formato de CEP inválido
  ```json
  "invalid zipcode"
  ```
- `404 Not Found`: CEP não encontrado no banco de dados
  ```json
  "can not find zipcode"
  ```

## 🛠️ Desenvolvimento Local

### Pré-requisitos

- Go 1.21 ou superior
- Docker
- Chave da WeatherAPI (gratuita em https://www.weatherapi.com/)

### Configuração

1. Clone o repositório:
```bash
git clone <repository-url>
cd CloudRun
```

2. Instale as dependências:
```bash
go mod tidy
```

3. Configure sua chave da WeatherAPI como variável de ambiente:

PowerShell:
```powershell
$env:WEATHER_API_KEY="SUA_CHAVE_AQUI"
```

Bash:
```bash
export WEATHER_API_KEY="SUA_CHAVE_AQUI"
```

### Executando Localmente

1. Usando Go diretamente:
```bash
go run main.go
```

2. Usando Docker:
```bash
docker build -t weather-service .
docker run -p 8080:8080 weather-service
```

O servidor será iniciado em `http://localhost:8080`

### Testes

Execute os testes automatizados:
```bash
go test -v
```

## 🐳 Implantação Docker

### Construa a imagem Docker:
```bash
docker build -t weather-service .
```

### Execute o contêiner:
```bash
docker run -p 8080:8080 weather-service
```

## ☁️ Implantação no Google Cloud Run

### Pré-requisitos

- Google Cloud SDK instalado e configurado
- gcloud CLI autenticado
- Projeto Google Cloud com API Cloud Run habilitada

### Passos de Implantação

1. Construa e envie para o Artifact Registry (via Cloud Build):
```bash
# Defina seu ID de projeto
export PROJECT_ID=your-project-id

# Crie um repositório Docker no Artifact Registry (uma vez)
gcloud artifacts repositories create weather-service \
  --repository-format=docker \
  --location=us-central1

# Construa a imagem
gcloud builds submit --tag us-central1-docker.pkg.dev/$PROJECT_ID/weather-service/weather-service:latest

# Implante no Cloud Run
gcloud run deploy weather-service \
  --image us-central1-docker.pkg.dev/$PROJECT_ID/weather-service/weather-service:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --set-env-vars WEATHER_API_KEY=your-api-key
```

2. Obtenha a URL do serviço:
```bash
gcloud run services describe weather-service \
  --platform managed \
  --region us-central1 \
  --format 'value(status.url)'
```

### Variáveis de Ambiente

Para implantação em produção, configure a chave da WeatherAPI como variável de ambiente:

```bash
gcloud run deploy weather-service \
  --image us-central1-docker.pkg.dev/$PROJECT_ID/weather-service/weather-service:latest \
  --set-env-vars WEATHER_API_KEY=your-api-key \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

## 🧪 Exemplos de Teste

### CEP Válido:
```bash
curl "http://localhost:8080/weather?cep=01153000"
```

### Formato de CEP Inválido:
```bash
curl "http://localhost:8080/weather?cep=123"
# Retorna: "invalid zipcode" (422)
```

### CEP Inexistente:
```bash
curl "http://localhost:8080/weather?cep=99999999"
# Retorna: "can not find zipcode" (404)
```

## 📊 Arquitetura

```
Requisição do Cliente → Handler HTTP → Validação CEP → API ViaCEP → API WeatherAPI → Conversão de Temperatura → Resposta JSON
```

## 🔧 Dependências

- `net/http`: Servidor e cliente HTTP
- `encoding/json`: Marshaling/unmarshaling JSON
- `regexp`: Validação de formato CEP
- ViaCEP API: Consulta de CEP brasileiro
- WeatherAPI: Provedor de dados meteorológicos

## 📝 Licença

Este projeto tem fins educacionais como parte do curso Go Expert.
