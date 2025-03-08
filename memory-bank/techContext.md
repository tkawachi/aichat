# Tech Context

## Technologies used
- **Go:** Programming language for the CLI tool
- **getopt:** Minimal dependency CLI parsing
- **Native Configuration:** Environment variables and YAML config
- **OpenAI API:** GPT-3/GPT-4 integration

## Development setup
- **Go 1.21+:** Required for modern features
- **Standard Library Focus:** Minimal external dependencies
- **Testing:** Native Go testing framework
- **Tooling:** Go modules for dependency management

## Technical constraints
- **API Rate Limits:**  LLM API rate limits and usage quotas.
- **Latency:** Network latency in API calls.
- **Error Handling:** Robust error handling for API communication and service disruptions.
- **Security:** Secure storage and handling of API keys.

## Dependencies
- github.com/pborman/getopt/v2
- github.com/sashabaranov/go-openai v1.38.0
- github.com/samber/go-gpt-3-encoder
- github.com/samber/lo v1.49.1
- github.com/dlclark/regexp2 v1.11.5
- golang.org/x/text v0.23.0
