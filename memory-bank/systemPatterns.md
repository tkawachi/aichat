# System Patterns

## System Architecture
- **Command Processing:** getopt-based parser for minimal dependency CLI handling
- **LLM API Interaction:** Direct OpenAI API integration with modular design for future expansion
- **Configuration Management:** Environment variable first approach with fallback to YAML config
- **Error Handling:** Centralized error handling to manage API errors, invalid commands, and other issues.

## Key technical decisions
- **Language Choice:** Go is chosen for its efficiency, concurrency, and ভাল ecosystem for CLI tools.
- **API Client Design:** Use of interfaces to abstract LLM API interactions, allowing for easy addition of new LLM providers.
- **Configuration Security:**  Storing API keys as environment variables and using secure configuration practices.

## Design patterns in use
- **Pipeline Pattern:** For processing input through tokenization and API interaction
- **Builder Pattern:** For constructing complex API request objects
- **Explicit Error Handling:** Using Go's native error wrapping and checking

## Component relationships
- **Command Parser** -> **LLM API Client** -> **LLM API**
- **Configuration Manager** -> **Command Parser** & **LLM API Client**
- **Error Handler** -> All components
