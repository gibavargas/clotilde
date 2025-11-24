# Import Clotilde Shortcut

## Quick Import Instructions

The `Clotilde.shortcut` file has been updated with your actual service URL and API key. However, Apple Shortcuts use a binary format that may require manual setup.

## Option 1: Manual Setup (Recommended)

Since Apple Shortcut files are complex binary formats, it's more reliable to create the shortcut manually in the Shortcuts app:

1. **Open Shortcuts app** on your iPhone
2. **Tap the +** button to create a new shortcut
3. **Name it**: "Clotilde"
4. **Add these actions in order**:

### Action 1: Dictate Text
- Search for "Dictate Text"
- Language: **Portuguese (Brazil)**
- Show Confirmation: **Off**

### Action 2: Get Contents of URL
- Search for "Get Contents of URL"
- Method: **POST**
- URL: `YOUR_SERVICE_URL/chat` (get from Cloud Run service description)
- Headers:
  - `Content-Type`: `application/json`
  - `X-API-Key`: `YOUR_API_KEY` (get from Secret Manager: `gcloud secrets versions access latest --secret=clotilde-api-key`)
- Request Body: **JSON**
  ```json
  {
    "message": "[Dictated Text]"
  }
  ```
  (Replace `[Dictated Text]` with the variable from Action 1)

### Action 3: Get Dictionary from Input
- Search for "Get Dictionary from Input"
- Input: (automatically from previous action)

### Action 4: Get Value for Key
- Search for "Get Value for Key"
- Key: `response`
- Dictionary: (automatically from previous action)

### Action 5: Speak Text
- Search for "Speak Text"
- Language: **Portuguese (Brazil)**
- Rate: **0.5** (slower for driving)
- Text: (automatically from previous action)

## Configure Shortcut Settings

1. Tap the **...** on the shortcut
2. Tap the **Settings** icon (gear)
3. Enable:
   - ✅ **Show in CarPlay**
   - ✅ **Show on Apple Watch** (optional)
4. Tap **Add to Siri**
5. Record: **"Falar com Clotilde"**
6. Tap **Done**

## Test

Say: **"Hey Siri, Falar com Clotilde"**

Then ask a question in Portuguese, for example:
- "Qual é a temperatura hoje?"
- "Como está o trânsito?"
- "Toca uma música"

## Troubleshooting

If the shortcut doesn't work:
1. Check your internet connection
2. Verify the API key is correct (get from Secret Manager)
3. Verify the URL is correct (get from Cloud Run service description)
4. Check that "Show in CarPlay" is enabled
5. Restart your iPhone and reconnect to CarPlay

## Service Information

- **Service URL**: Get from Cloud Run: `gcloud run services describe clotilde --region=us-central1 --format="value(status.url)"`
- **API Key**: Get from Secret Manager: `gcloud secrets versions access latest --secret=clotilde-api-key`
- **Status**: ✅ Deployed and running

