# Optional template extensions

These modules are independent implementations. They do not contain source copied from the ShipAny reference project. No additional payment provider or dependency is introduced.

## Data tables

`components/data-table.tsx` exposes `DataTable<T>` with typed columns, stable row keys, a caption, loading/empty/error states, optional controlled sorting, and server pagination. Column cells return React nodes; do not provide HTML strings for injection. Sorting calls `onSort`; the host must sort the complete dataset or request the sorted backend page. It never sorts one page and presents it as the complete dataset.

Credit history at `/credits` uses this component. Pass translated labels with `useI18n`, including loading and pagination labels. Keep the API's actual page limit in the pagination props.

## Forms

`components/form-field.tsx` exposes `FormField` and `FormActions`. A field links its label, optional description and error to one native control. Required controls retain native validation. `FormActions` disables submission while busy and displays the shared pending indicator. Use these primitives when adding your own product forms. Request, auth and domain validation remain in their original API/service layers.

## Google Analytics 4

Set `NEXT_PUBLIC_GA4_MEASUREMENT_ID` to a public `G-...` measurement ID and rebuild. Blank or malformed values keep analytics disabled. A visitor preference appears on public pages. Declining loads no Google script. The visitor can reopen the preference and disable analytics; revocation reloads the current page to remove the loaded script.

Only public page views are sent by this integration. URLs omit all query parameters and fragments. It does not send user IDs, emails, chat text, or payment amounts. Workspace, sign-in, callback and admin routes disable collection.

**Before enabling:** in the GA4 web stream, disable Enhanced Measurement's automatic pageviews/history-based page changes and other automatic measurements if you require this integration's public-only, query-free event policy. The integration sets `send_page_view: false` and sends its own pageview; account-level enhanced measurement must not independently send history events. Configure your own disclosures and data practices for the deployed product.

Reference: https://developers.google.com/analytics/devguides/collection/ga4/views

## Crisp support

Set `NEXT_PUBLIC_CRISP_WEBSITE_ID` to your public workspace UUID and rebuild. Blank or malformed values keep the support launcher disabled. Opening the launcher mounts Crisp's documented external chat URL in a cross-origin iframe, inside the shared responsive dialog. Closing it removes the iframe. A new-window fallback is also available. The integration does not load Crisp's script into the host page or automatically pass the signed-in identity, email, messages, URL query or referrer. Crisp's own chat/session behavior is controlled by your Crisp workspace and the browser.

Reference: https://help.crisp.chat/en/article/how-to-embed-the-crisp-live-chat-chatbox-in-an-iframe-bkfh98/

## AI integration

`foundation/aix` is a reusable, stateless model adapter. `templates/quickstart/internal/ai` maps the backend configuration to that adapter. Neither supplies a customer chat product, HTTP routes, conversation storage, quotas or credit charging. The customer area contains account, subscription/billing and referral services, with credit history as an auxiliary page. No note or upload UI is supplied; the backend implementation references and shared integration capabilities remain available.

Configure the **backend**:

```dotenv
APP_HOST_AI_ENABLED=true
APP_HOST_AI_BASE_URL=https://api.openai.com/v1
APP_HOST_AI_API_KEY=your-server-side-provider-key
APP_HOST_AI_MODEL=your-compatible-chat-model
APP_HOST_AI_MAX_OUTPUT_TOKENS=1024
```

Never put the key in a `NEXT_PUBLIC_*` variable. Enabling configuration validates it at startup; it does not register an endpoint. In your product composition code, use the injected client:

```go
provider, err := ai.New(deps.Config.AI, deps.Modules.Config.HTTPClient)
// Handle err (including ai.ErrDisabled) here, then inject provider into your service.
result, err := provider.Complete(ctx, []aix.Message{
    {Role: "system", Content: "Summarize the supplied text."},
    {Role: "user", Content: input},
})
```

The snippet uses `internal/ai` from the backend template and `github.com/brizenchi/go-modules/foundation/aix`. Pass the request context; a product service decides authorization and input limits before calling it. See [the adapter README](../../foundation/aix/README.md) for complete setup, transport bounds and error handling.

The adapter sends one non-streaming Chat Completions-compatible request and adds no retries. The selected provider/model must support `model`, `messages`, `max_tokens` and `stream: false`. Your injected transport may have its own retry behavior. No user identity or database is needed by the adapter, and it does not log or persist prompts or output.

Your product owns conversation history, prompts, task status, rate limits and billing. The credit module remains reusable; reservation, charging, refund and idempotency belong to the business operation that delivers the result. For a chat product, build its conversation model and UI there. For a summarizer or another tool, call the same adapter from that feature without creating a chat product.

Previously created demo chat tables are not dropped, but are no longer migrated or used. No real-provider call is made by the isolated preview or the test suite.
