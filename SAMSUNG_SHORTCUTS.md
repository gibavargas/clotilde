# 🤖 Using Clotilde with Samsung Galaxy Shortcuts

A complete guide to integrate Clotilde AI Assistant with Samsung Galaxy devices using Bixby Routines, HTTP Shortcuts, and Android Auto.

---

## 📱 Supported Android Automation Apps

Clotilde works with multiple Android automation tools:

1. **Bixby Routines** (Built-in Samsung)
2. **HTTP Shortcuts** (Recommended - Free on Play Store)
3. **Tasker** (Advanced automation)
4. **MacroDroid** (User-friendly)
5. **Automate** (Visual flow-based)

---

## 🚀 Quick Start with HTTP Shortcuts (Recommended)

### Step 1: Install HTTP Shortcuts

Download from Google Play Store:
- **App**: HTTP Shortcuts by Roland Kluge
- **Free**: Yes
- **Link**: https://play.google.com/store/apps/details?id=ch.rmy.android.http_shortcuts

### Step 2: Create Your First Shortcut

1. Open **HTTP Shortcuts** app
2. Tap **"+"** to create new shortcut
3. Configure the shortcut:

**Basic Settings:**
```
Name: Ask Clotilde
Description: AI Assistant
Icon: Choose any icon
```

**Request Settings:**
```
Method: POST
URL: https://clotilde-zxymv6mlja-uc.a.run.app/chat
```

**Headers:**
```
Content-Type: application/json
X-API-Key: <YOUR_API_KEY>
```

**Request Body:**
```json
{
  "message": "{input}"
}
```

**Response Handling:**
```
Response Type: JSON
Response Key: response
Display: Show in dialog or speak
```

### Step 3: Test Your Shortcut

1. Tap the shortcut
2. Enter a question: "What is the capital of Brazil?"
3. Clotilde will respond!

---

## 🚗 Android Auto Integration

### Option 1: Using Android Auto with Google Assistant

1. Enable Google Assistant in Android Auto
2. Create a custom routine in Google Assistant app:
   - Open Google app → Settings → Google Assistant → Routines
   - Add custom phrase: "Ask Clotilde [question]"
   - Action: Use HTTP Shortcuts or Tasker plugin

### Option 2: Using HTTP Shortcuts with Android Auto

1. Install **HTTP Shortcuts** (supports Android Auto)
2. Create shortcut as described above
3. Enable Android Auto support in HTTP Shortcuts settings
4. Shortcut will appear in Android Auto launcher

**Note**: Response will be spoken via text-to-speech

---

## 🎯 Advanced Setup with Bixby Routines

Bixby Routines can trigger HTTP requests, but requires additional setup.

### Step 1: Install Bixby Routines Helper

You'll need a helper app like **Tasker** or **HTTP Shortcuts** to make HTTP requests.

### Step 2: Create Bixby Routine

1. Open **Bixby Routines** app
2. Tap **"+"** to create new routine
3. Configure:

**IF (Trigger):**
```
- Voice command: "Ask Clotilde"
- Button press
- Time-based
- Location-based
```

**THEN (Action):**
```
- Run HTTP Shortcuts shortcut
- Or run Tasker task
```

---

## 📋 Complete API Reference

### Endpoint
```
POST https://clotilde-zxymv6mlja-uc.a.run.app/chat
```

### Headers
```http
Content-Type: application/json
X-API-Key: YOUR_API_KEY_HERE
```

### Request Body
```json
{
  "message": "Your question here"
}
```

### Response (Success)
```json
{
  "response": "Clotilde's answer here"
}
```

### Response (Error)
```json
{
  "error": "Error message here"
}
```

### HTTP Status Codes
- **200 OK**: Success
- **400 Bad Request**: Invalid input
- **401 Unauthorized**: Missing or invalid API key
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error

---

## 🔧 Setup Examples for Different Apps

### Example 1: Tasker Setup

1. Create new Task
2. Add Action → Net → HTTP Request
3. Configure:

```
Method: POST
URL: https://clotilde-zxymv6mlja-uc.a.run.app/chat
Headers:
  Content-Type: application/json
  X-API-Key: YOUR_API_KEY

Body:
  {"message": "%input"}

Response Variable: %response
```

4. Add Action → Say → Speak Text
```
Text: %response
```

### Example 2: MacroDroid Setup

1. Create new Macro
2. **Trigger**: Your choice (voice, button, etc.)
3. **Action**: HTTP Request
   - Method: POST
   - URL: https://clotilde-zxymv6mlja-uc.a.run.app/chat
   - Headers: Content-Type: application/json, X-API-Key: YOUR_KEY
   - Body: {"message": "[prompt:Enter question]"}
4. **Action**: Speak Text
   - Text: {json_read,response}

### Example 3: Automate (Flow-based)

1. Create new flow
2. Add "HTTP request" block:
   - Method: POST
   - URL: https://clotilde-zxymv6mlja-uc.a.run.app/chat
   - Content type: application/json
   - Custom headers: X-API-Key: YOUR_KEY
   - Request content: {"message": "What's the weather?"}
3. Add "Text to speech" block
   - Read from: HTTP response → response

---

## 🎤 Voice Commands Setup

### With Google Assistant

1. Open Google app → Settings → Google Assistant → Routines
2. Create new routine:

**When I say:**
```
"Ask Clotilde [question]"
or
"Hey Clotilde [question]"
```

**My Assistant should:**
```
- Open HTTP Shortcuts shortcut "Ask Clotilde"
- Pass [question] as input
```

### With Bixby Voice

1. Open Bixby app
2. Create Quick Command:

**When I say:**
```
"Ask Clotilde [question]"
```

**Run:**
```
- Launch HTTP Shortcuts
- Run "Ask Clotilde" shortcut
- With input: [question]
```

---

## 🚀 Performance Tips for Android Auto

### Reduce Response Time

1. **Use Fast Models**: Clotilde defaults to Claude Haiku 4.5 (1-3s response)
2. **Keep Questions Simple**: Shorter questions = faster responses
3. **Avoid Web Search**: Factual questions are faster than web searches
4. **Check Network**: Use LTE/5G, not slow 3G

### Expected Response Times

| Query Type | Expected Time | Model Used |
|------------|---------------|------------|
| Simple factual | 2-4 seconds | Claude Haiku 4.5 |
| Complex analysis | 4-8 seconds | Claude Haiku 4.5 |
| Web search | 9-12 seconds | Perplexity + Claude |
| Math/calculations | 2-4 seconds | Claude Haiku 4.5 |

**Note**: Android Auto has ~30s timeout (same as Apple CarPlay)

---

## 🔒 Security Best Practices

### Protect Your API Key

1. **Never share your API key** in public forums or screenshots
2. **Store securely** in automation app (encrypted storage)
3. **Use device lock** to prevent unauthorized access
4. **Rotate keys** if compromised

### Recommended Setup

- Use **HTTP Shortcuts** or **Tasker** (support encrypted storage)
- Enable device PIN/fingerprint lock
- Don't share shortcuts containing API keys

---

## 🐛 Troubleshooting

### Common Issues and Solutions

#### Issue: "Unauthorized" (401)
**Solution**: Check your API key
```
1. Verify X-API-Key header is set
2. Check for typos in API key
3. Ensure no extra spaces in header value
```

#### Issue: "Timeout" or "No Response"
**Solution**: Network or server issue
```
1. Check internet connection (LTE/5G/WiFi)
2. Verify URL is correct
3. Try simpler question first
4. Check Clotilde service status
```

#### Issue: "Invalid Request" (400)
**Solution**: Check request format
```
1. Verify Content-Type: application/json header
2. Check JSON syntax in body
3. Ensure "message" field is present
```

#### Issue: "Rate Limited" (429)
**Solution**: Too many requests
```
1. Wait 1 minute before retrying
2. Reduce request frequency
3. Contact admin for rate limit increase
```

#### Issue: App Won't Speak Response
**Solution**: Enable text-to-speech
```
1. Install Google Text-to-Speech
2. Enable TTS in automation app settings
3. Grant microphone/speech permissions
4. Test TTS with simple text first
```

---

## 📱 Recommended Android Auto Workflow

### Step-by-Step Setup

1. **Install HTTP Shortcuts** from Play Store
2. **Create "Ask Clotilde" shortcut** (see Quick Start)
3. **Enable Android Auto support**:
   - HTTP Shortcuts → Settings → Android Auto → Enable
4. **Add to Android Auto launcher**:
   - Android Auto → Settings → Customize launcher
   - Add HTTP Shortcuts to launcher
5. **Test in car**:
   - Connect phone to car
   - Open Android Auto
   - Find HTTP Shortcuts in launcher
   - Tap "Ask Clotilde"
   - Speak your question
   - Listen to response

---

## 🎯 Example Use Cases

### Use Case 1: Navigation Help
```
You: "What's the fastest route to São Paulo avoiding tolls?"
Clotilde: "A rota mais rápida sem pedágios é pela BR-116..."
```

### Use Case 2: News Updates
```
You: "What are the latest news in Brazil?"
Clotilde: "As principais notícias de hoje incluem..."
```

### Use Case 3: Weather Check
```
You: "What's the weather in Rio tomorrow?"
Clotilde: "Amanhã em Rio de Janeiro, a previsão é..."
```

### Use Case 4: Quick Facts
```
You: "Who won the last World Cup?"
Clotilde: "A Argentina venceu a Copa do Mundo de 2022..."
```

### Use Case 5: Calculations
```
You: "How much is 1500 reais in dollars?"
Clotilde: "1500 reais equivalem a aproximadamente..."
```

---

## 🌟 Advanced Features

### Custom Response Formatting

You can request specific response formats:

**JSON Response:**
```json
{
  "message": "List 3 restaurants in São Paulo. Respond in JSON format."
}
```

**Bullet Points:**
```json
{
  "message": "Give me 5 tips for driving safely. Use bullet points."
}
```

**Short Answers:**
```json
{
  "message": "What's the capital of France? Answer in one word."
}
```

---

## 📞 Support and Help

### Getting Your API Key

Contact your Clotilde administrator to get an API key.

### Service URL

Current service URL:
```
https://clotilde-zxymv6mlja-uc.a.run.app
```

**Note**: This URL may change. Check with your admin for the latest endpoint.

### Admin Dashboard

Admins can monitor usage and adjust settings at:
```
https://clotilde-zxymv6mlja-uc.a.run.app/admin/
```

---

## 🔄 Updates and Changes

### Version History

**v2.0** - December 2025
- ✅ Added Claude Haiku 4.5 support (FAST responses)
- ✅ Improved Android Auto compatibility
- ✅ Reduced response time to 2-4 seconds
- ✅ Enhanced Portuguese language support

**v1.5** - November 2025
- Added Perplexity web search integration
- Improved error handling
- Added rate limiting

---

## 💡 Tips and Tricks

### Optimize for Speed

1. **Keep questions concise**: "Weather tomorrow?" instead of "Can you please tell me what the weather will be like tomorrow in my location?"
2. **Use fast models**: Default is already optimized (Claude Haiku 4.5)
3. **Avoid complex analysis**: Save deep questions for when not driving
4. **Prefer facts over searches**: "Capital of France?" is faster than "Latest news in France?"

### Improve Accuracy

1. **Be specific**: "Weather in São Paulo tomorrow" vs "Weather tomorrow"
2. **Use proper nouns**: "São Paulo" vs "the big city"
3. **Specify language**: Clotilde auto-detects, but you can specify: "Respond in English"
4. **Follow up**: You can reference previous questions in the same session

### Battery Saving

1. **Disable auto-refresh** in HTTP Shortcuts
2. **Use on-demand shortcuts** instead of background monitoring
3. **Limit web search queries** (they consume more battery)

---

## 📝 Sample Shortcuts Collection

### Download Ready-to-Use Shortcuts

Coming soon: Pre-configured HTTP Shortcuts exports for:
- ✅ Basic Q&A
- ✅ Weather queries
- ✅ News updates
- ✅ Navigation help
- ✅ Quick calculations

---

## 🤝 Contributing

Found a better way to integrate with Samsung/Android?
Share your setup with the community!

---

**Last Updated**: December 23, 2025  
**Status**: Active  
**Android Version**: Compatible with Android 8.0+  
**Samsung One UI**: Compatible with One UI 3.0+  

---

**Made with ❤️ for Samsung Galaxy and Android Auto users**

