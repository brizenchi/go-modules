# aix

Stateless server-side text generation using the Chat Completions JSON protocol. This is an integration adapter, not a chat product. It has no dependency on business modules, storage, identities or billing.

```go
client, err := aix.New(aix.Config{
    APIKey: providerKey,
    Model: compatibleModel,
}, tracedHTTPClient)
if err != nil { return err }
result, err := client.Complete(ctx, []aix.Message{
    {Role: "system", Content: "Summarize the supplied text."},
    {Role: "user", Content: sourceText},
})
```

Configuration requires an HTTPS API base, secret and compatible model. The base defaults to `https://api.openai.com/v1`; output defaults to 1024 tokens, capped at 4096. The model must support `messages`, `max_tokens` and `stream: false`. Inputs are bounded to 128 text messages, 64 KB per message and 256 KB total; responses are limited to 2 MiB. Each call inherits context and has a maximum 45-second deadline. Redirects are refused and upstream response bodies/transport details are not included in errors. `errors.Is` distinguishes invalid configuration, invalid input and provider failures; cancellation remains detectable.

The adapter adds no retries. The supplied transport must have an appropriate retry policy for billable requests. A product owns authentication, resource ownership, conversation/history storage, prompts, limits, idempotency, usage-based charging and compensation. Keep provider keys on the server and handle each error once at the product boundary.

Protocol reference: [Chat Completions](https://developers.openai.com/api/reference/resources/chat).
