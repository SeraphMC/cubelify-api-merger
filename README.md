# API Merger

This application merges responses from multiple Cubelify formatted APIs into a single response.

## Configuration

The application's behaviour is configured using a JSON file (e.g., `config.json`). This file defines the API endpoints to be called and how their responses should be handled. A working configuration is included in the project repository under "config.json".

In the Cubelify overlay, simply paste this line in to your Custom API URL:

```text
http://localhost:3000/merger?source={{source}}&id={{id}}&name={{name}}
```

### `config.json` Structure

The `config.json` file contains a JSON object where each key represents a custom name for an API configuration.  Each of these configurations is an object with the following structure:

```json
{
  "example-api": {
    "url": "https://example.com/tags",
    "querystring": {
      "token": "my-token",
      "api": "v2",
      "exampleparam": "a-random-example-string",
      "more-keys": "and-another-example-string",
      "code": 123
    },
    "request_params": {
      "source": "remapped-source",
      "id": "playerId",
      "name": "playerName"
    }
  }
}
```
Example result: https://example.com/tags?api=v2&code=123&exampleparam=a-random-example-string&more-keys=and-another-example-string&playerId=5719c127-ffa6-4fe3-a17c-16247c836662&playerName=Endure&token=my-token

custom_name:  A user-defined name for this API configuration (e.g: example-api). This name is used in the application's output to identify the source of a response.

url:  The URL of the API endpoint to be called.

querystring: (Optional) A JSON object representing the query string parameters to be included in the API request.

Each key-value pair in this object will be added as a query parameter to the URL.

request_params: (Optional) A JSON object that defines how incoming request parameters from the user's request to the merger should be remapped before being sent to the target API.

Each key-value pair defines a mapping:

The key is the name of the parameter expected in the incoming request to the merger.

The value is the name of the parameter that should be used when making the request to the target API.

Detailed Explanation of request_params

The request_params section provides a way to rename or remap parameters from the user's request to the names expected by the target API.  For example, consider this configuration:
```json
{
  "example-api": {
    "url": "https://example.com/tags",
    "request_params": {
      "source": "remapped-source",
      "id": "playerId",
      "name": "playerName"
    }
  }
}
```

In this case:

If a user makes a request to the merger with a query parameter named source, the merger will pass that parameter to https://example.com/tags, but it will be named remapped-source.

If the incoming request has a parameter named id, it will be passed to the target API as playerId.

The incoming parameter name will be passed as playerName.

This is useful when the API you are merging has different parameter names than the ones you want to expose to your users.  If a parameter is not listed in request_params, it will be passed to the target API with its original name.

This was rewritten from my original README using Google Gemini to ensure readability for non-technical users.
