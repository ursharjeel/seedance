# CAPTURE LIMITATIONS AND EVIDENCE QUALITY

## What We Actually Captured (HIGH CONFIDENCE)

### ✅ Proven by Direct Evidence

1. **API Endpoint**
   - `POST https://api.leonardo.ai/v1/graphql`
   - Confirmed: 180 GraphQL POST requests captured

2. **Model Identifier**
   - `seedance-2.5` exists
   - Evidence: CDN URLs show `bytedance/seedance-2.5_*.mp4` and `.jpg`

3. **Request Structure (Shape Only)**
   - GraphQL mutation with `operationName`, `variables`, `query`
   - Variables contain: `request.model`, `request.public`, `request.parameters`
   - Parameters shape includes: `prompt`, `duration`, `height`, `width`, `quantity`, `seed`, `motion_has_audio`

4. **Response Evidence**
   - At least 2 completed generations (two unique generation IDs in CDN URLs)
   - Video files successfully generated at CDN
   - Thumbnails (.jpg) generated alongside videos

5. **Authentication**
   - `Authorization` header present (value redacted)
   - `x-leo-schema-version` header exists

---

## What We Inferred (MEDIUM CONFIDENCE)

### ⚠️ Based on leo2api Source Code Analysis

These are NOT from the capture, but from reading existing Seedance 2.0 code:

1. **GraphQL Mutation Text**
   ```graphql
   mutation Generate($request: CreateGenerationRequest!) {
     generate(request: $request) {
       apiCreditCost
       generationId
       __typename
     }
   }
   ```
   - Source: `client.go` line 797
   - Assumption: Seedance 2.5 uses same mutation as 2.0

2. **Default Parameters**
   - width: 1280, height: 720
   - duration: 4 seconds
   - seed: -1 (random)
   - Source: leo2api defaults for Seedance 2.0
   - Assumption: 2.5 inherits 2.0 defaults

3. **Status Polling**
   - Query: `GetAIGenerationFeedStatuses`
   - Status values: `PENDING`, `COMPLETE`, `FAILED`
   - Source: leo2api existing code
   - Assumption: Same polling mechanism

4. **Result Retrieval**
   - Field: `motionMP4URL` in `generated_images`
   - Query: `GetGenerationDetail`
   - Source: leo2api existing code
   - Assumption: Same response structure

---

## What We Don't Know (LOW/NO EVIDENCE)

### ❌ Not Proven by Capture

1. **Exact Parameter Values**
   - We sanitized actual values to protect credentials
   - Don't know the exact prompt, duration, or dimensions used in captured generations

2. **Parameter Constraints**
   - Valid resolution ranges (is 1280x720 the only option? Can it do 1920x1080?)
   - Valid duration ranges (4 seconds only? 4-15 like other models?)
   - Mode values (we assume `RESOLUTION_720` but didn't capture it)

3. **prompt_enhance Field**
   - Mentioned in leo2api code
   - Not confirmed in our capture schema

4. **Credit Cost**
   - `apiCreditCost` field exists in mutation response
   - We didn't capture the actual response body values

5. **Error Handling**
   - Don't know exact error messages
   - Don't know failure modes specific to 2.5

6. **Advanced Features**
   - Image/video reference support
   - Audio reference support
   - Start/end frame support
   - (These exist in 2.0 but not confirmed for 2.5)

---

## Capture Script Limitations

### Why We Lost Data

The sanitization was **too aggressive** by design (to protect credentials):

```python
def sanitize_body(body_text, content_type):
    if 'json' in content_type.lower():
        data = json.loads(body_text)
        # ❌ We only kept the shape, not values
        return {"_shape": describe_json_shape(data), ...}
```

**What this means:**
- We can see there's a `prompt` field
- We can't see what the prompt value was
- We can see there's a `duration` field
- We can't see if it was 4, 5, 10, or 15 seconds

### What We Should Have Done

1. **Keep parameter values** (they're not credentials)
2. **Keep mutation query text** (it's not sensitive)
3. **Keep response bodies** (generation IDs are not secrets after the fact)
4. **Only redact**: Authorization tokens, cookies, user IDs

---

## Go Example Code Status

### ⚠️ Code is Untested

The `seedance25_example.go` file:

- **Based on:** leo2api Seedance 2.0 implementation
- **Assumption:** 2.5 works identically to 2.0
- **Reality:** Not compiled, not run, not verified
- **May fail on:** 
  - Parameter validation differences
  - Response structure changes
  - New required fields
  - Different error messages

**To verify:**
1. You must test it with a real Leonardo account
2. You may need to adjust parameters
3. You may need to handle new error cases

---

## Recommendations

### To Get Complete Evidence

**Option 1: Manual Chrome DevTools Capture**
1. Open Leonardo.ai
2. F12 → Network tab → Filter: `graphql`
3. Generate a Seedance 2.5 video
4. Find the `generate` request
5. Copy → Copy as cURL (keeps real values)
6. Copy → Payload tab (see exact JSON)

**Option 2: Re-run Capture Without Sanitization**
```python
# Modified version that keeps safe values
def sanitize_body(body_text, content_type):
    data = json.loads(body_text)
    # Keep everything except auth/cookies/user_id
    return data  # Then manually redact only sensitive fields
```

**Option 3: Browser Extension**
- Use browser extension to export HAR file
- HAR preserves full request/response
- Manually review and redact sensitive data

---

## What the Documentation Should Say

### Current Claims (Too Strong)

❌ "Complete API reference"
❌ "Exact parameters"
❌ "Working code example"
❌ "All you need to implement"

### Honest Claims (Accurate)

✅ "Evidence that Seedance 2.5 exists and uses GraphQL"
✅ "Parameter structure inferred from Seedance 2.0"
✅ "Untested code example based on 2.0 patterns"
✅ "Starting point for implementation (requires testing)"

---

## Confidence Levels Summary

| Item | Evidence Level | Source |
|------|---------------|--------|
| Model exists | ✅ HIGH | CDN URLs |
| Uses GraphQL | ✅ HIGH | Network capture |
| Endpoint URL | ✅ HIGH | Network capture |
| Parameter names | ✅ HIGH | Capture schema |
| GraphQL mutation text | ⚠️ MEDIUM | leo2api code |
| Default values | ⚠️ MEDIUM | leo2api code |
| Valid ranges | ❌ LOW | Assumption |
| Mode values | ❌ LOW | Assumption |
| Credit cost | ❌ UNKNOWN | Not captured |
| Go example works | ❌ UNTESTED | Not run |

---

## Bottom Line

**What we proved:**
- Seedance 2.5 exists
- It uses the same GraphQL endpoint as 2.0
- Parameter structure is similar to 2.0
- Videos successfully generate

**What we didn't prove:**
- Exact parameter values and constraints
- Response structure (assumed from 2.0)
- Error handling specifics
- Code example functionality

**What you should do:**
1. Use the documentation as a **starting point**
2. **Test everything** with your Leonardo account
3. **Expect to debug** parameter values
4. **Update the documentation** with real findings

---

## Suggested Next Steps

1. **Manual capture** with Chrome DevTools (keeps real values)
2. **Test the Go example** with a real account
3. **Document actual behavior** you observe
4. **Update this repository** with verified data
5. **Add a TESTING.md** with real test results

This is a **research snapshot**, not a production-ready specification.
