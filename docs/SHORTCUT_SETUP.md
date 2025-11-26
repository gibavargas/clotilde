# Apple Shortcut Setup Guide

## Quick Setup

Since Apple Shortcut files are binary/plist format, you'll need to create the shortcut manually in the Shortcuts app. Follow these steps:

## Step-by-Step Instructions

### 1. Open Shortcuts App

On your iPhone, open the **Shortcuts** app.

### 2. Create New Shortcut

1. Tap the **+** button in the top right
2. Name it **"Clotilde"**
3. Tap **Add Action**

### 3. Add Actions (In Order)

#### Action 1: Dictate Text

1. Search for **"Dictate Text"**
2. Select **"Dictate Text"**
3. Configure:
   - Language: **Portuguese (Brazil)**
   - Show Confirmation: **Off** (for faster CarPlay use)

#### Action 2: Get Contents of URL

1. Search for **"Get Contents of URL"**
2. Select **"Get Contents of URL"**
3. Configure:
   - **Method**: POST
   - **URL**: `YOUR_SERVICE_URL/chat` (get from: `gcloud run services describe clotilde --region=us-central1 --format="value(status.url)"`)
   - **Headers**:
     - Tap **Show More**
     - Add header:
       - Key: `Content-Type`
       - Value: `application/json`
     - Add header:
       - Key: `X-API-Key`
       - Value: `YOUR_API_KEY` (obtain from Secret Manager: `gcloud secrets versions access latest --secret=clotilde-api-key`)
   - **Request Body**: JSON
     - Tap **Request Body**
     - Select **JSON**
     - Tap the JSON field
     - Add:
       ```json
       {
         "message": "[Dictated Text]"
       }
       ```
     - Replace `[Dictated Text]` with the variable from the previous action:
       - Tap the `[Dictated Text]` placeholder
       - Select **"Dictated Text"** from the list

#### Action 3: Get Dictionary from Input

1. Search for **"Get Dictionary from Input"**
2. Select **"Get Dictionary from Input"**
3. The input should automatically be the previous action's output

#### Action 4: Get Value for Key

1. Search for **"Get Value for Key"**
2. Select **"Get Value for Key"**
3. Configure:
   - **Key**: `response`
   - **Dictionary**: Should automatically be the previous action's output

#### Action 5: Speak Text

1. Search for **"Speak Text"**
2. Select **"Speak Text"**
3. Configure:
   - **Language**: Portuguese (Brazil)
   - **Rate**: 0.5 (slower for better comprehension while driving)
   - **Text**: Should automatically be the previous action's output

### 4. Configure Shortcut Settings

1. Tap the **...** button on the shortcut
2. Tap the **Settings** icon (gear)
3. Enable:
   - **Show in CarPlay**
   - **Show on Apple Watch** (optional)
4. Tap **Add to Siri**
5. Record phrase: **"Falar com Clotilde"** (or your preferred phrase)
6. Tap **Done**

### 5. Test the Shortcut

1. Say **"Hey Siri, Falar com Clotilde"**
2. Wait for the dictation prompt
3. Say your question in Portuguese
4. Clotilde should respond

## Troubleshooting

### Shortcut doesn't appear in CarPlay

- Make sure "Show in CarPlay" is enabled in shortcut settings
- Restart your iPhone
- Reconnect to CarPlay

### "Invalid API key" error

- Verify the API key in Secret Manager matches the one in your shortcut
- Check the `X-API-Key` header is set correctly
- Ensure no extra spaces in the header value

### No response from Clotilde

- Check your internet connection
- Verify the service URL is correct
- Check Cloud Run service logs: `gcloud run services logs read clotilde --region us-central1`

### Response is in wrong language

- The system prompt is set to Brazilian Portuguese by default
- You can ask Clotilde to respond in another language if needed

## Advanced Configuration

### Custom Siri Phrase

You can set any Siri phrase you want:
- "Falar com Clotilde"
- "Clotilde"
- "Assistente"
- Or any phrase you prefer

### Adjust Speech Rate

In the "Speak Text" action, adjust the rate:
- 0.3 = Very slow
- 0.5 = Slow (recommended for driving)
- 0.7 = Normal
- 1.0 = Fast

### Error Handling

To add error handling:

1. After "Get Value for Key", add **"If"** action
2. Check if value is empty or contains "error"
3. If error, speak a friendly message like "Desculpe, ocorreu um erro. Tente novamente."
4. Otherwise, speak the response

## Security Notes

- **Never share your API key**: Keep it private
- **Use Secret Manager**: Store API keys securely
- **Rotate keys regularly**: Update every 90 days
- **Monitor usage**: Check Cloud Logging for unusual activity

