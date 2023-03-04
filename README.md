# aichat

[![Maintainability](https://api.codeclimate.com/v1/badges/a94fa8eb02349a9ca8da/maintainability)](https://codeclimate.com/github/tkawachi/aichat/maintainability)

This is a program to use OpenAI's ChatAPI from the command line.

You need to set the API Key in the OPENAI_API_KEY environment variable.

When executed, you can interact with it on the terminal.
To exit, type Ctl-D.

```
$ aichat
user: Hello!
assistant:

Hello there! How may I assist you today?
user: How will AI change the world in the future?
assistant:

As an AI language model, I can say that AI has the potential to transform virtually every aspect of our lives, from healthcare to education,
(omitted)
```

Also, you can use `aichat foo` command by putting the prompt template as `$HOME/.aichat/prompts/foo.yml`. You can replace the `foo` part with any name you like.
The contents of `foo.yml` should look like this

```yaml
messages:
  - role: system
    content: A brief description of the program, using only lowercase letters and hyphens, appropriate for the program. You may use up to three hyphens.
  - role: user
    content: $INPUT
```

Command line input is embedded in `$INPUT` and sent to the API.

To use the prompt above, do the following

```
$ aichat foo Command line program to utilize AI
ai-utilization-cli
```
