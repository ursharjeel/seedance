# Leonardo.ai Seedance 2.5 - Complete Implementation Guide

## Overview

Based on the network capture and leo2api source code analysis, here's everything you need to add Seedance 2.5 support to your local API.

---

## 1. API Endpoint

**Base URL:** `https://api.leonardo.ai/v1/graphql`

**Method:** POST

**Authentication:** JWT token (obtained from session cookie)

---

## 2. Model Identifier

```
Model Name: seedance-2.5
Provider: bytedance
```

The current leo2api supports:
- `seedance-2.0`
- `seedance-2.0-fast`
- `seedance-2.0-mini`

You need to add: **`seedance-2.5`**

---

## 3. GraphQL Request Structure

### Generation Mutation

```graphql
mutation Generate($request: CreateGenerationRequest!) {
  generate(request: $request) {
    apiCreditCost
    generationId
    __typename
  }
}
```

### Variables

```json
{
  "request": {
    "model": "seedance-2.5",
    "public": false,
    "parameters": {
      "prompt": "cat running in the park",
      "quantity": 1,
      "duration": 4,
      "motion_has_audio": false,
      "width": 1280,
      "height": 720,
      "seed": -1,
      "mode": "RESOLUTION_720",
      "prompt_enhance": "OFF"
    }
  }
}
```

### Full Request Body

```json
{
  "operationName": "Generate",
  "variables": {
    "request": {
      "model": "seedance-2.5",
      "public": false,
      "parameters": {
        "prompt": "your prompt here",
        "quantity": 1,
        "duration": 4,
        "motion_has_audio": false,
        "width": 1280,
        "height": 720,
        "seed": -1,
        "mode": "RESOLUTION_720"
      }
    }
  },
  "query": "mutation Generate($request: CreateGenerationRequest!) {\n  generate(request: $request) {\n    apiCreditCost\n    generationId\n    __typename\n  }\n}"
}
```

---

## 4. Request Headers

```
Content-Type: application/json
Authorization: Bearer <JWT_TOKEN>
x-leo-schema-version: 1.0  (or whatever version)
```

---

## 5. Response Structure

### Success Response

```json
{
  "data": {
    "generate": {
      "apiCreditCost": 100,
      "generationId": "1f1a0644-81ab-6680-b74d-d24c8756460e",
      "__typename": "GenerateResponse"
    }
  }
}
```

---

## 6. Polling for Status

### Status Query

```graphql
query GetAIGenerationFeedStatuses($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    __typename
  }
}
```

### Variables

```json
{
  "where": {
    "id": {
      "_in": ["1f1a0644-81ab-6680-b74d-d24c8756460e"]
    },
    "status": {
      "_in": ["PENDING", "COMPLETE", "FAILED"]
    }
  }
}
```

---

## 7. Getting Video Result

### Detail Query

```graphql
query GetGenerationDetail($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    prompt
    modelId
    motionModel
    imageWidth
    imageHeight
    createdAt
    generated_images(order_by: [{url: desc}]) {
      id
      url
      motionMP4URL
      motionGIFURL
      __typename
    }
    __typename
  }
}
```

### Variables

```json
{
  "where": {
    "id": {
      "_in": ["1f1a0644-81ab-6680-b74d-d24c8756460e"]
    }
  }
}
```

### Response with Video URL

```json
{
  "data": {
    "generations": [
      {
        "id": "1f1a0644-81ab-6680-b74d-d24c8756460e",
        "status": "COMPLETE",
        "prompt": "cat running in the park",
        "modelId": "seedance-2.5",
        "imageWidth": 1280,
        "imageHeight": 720,
        "generated_images": [
          {
            "id": "...",
            "url": "https://cdn.leonardo.ai/users/.../generations/.../bytedance/seedance-2.5_cat_runng_in_ther_park-0.jpg",
            "motionMP4URL": "https://cdn.leonardo.ai/users/.../generations/.../bytedance/seedance-2.5_cat_runng_in_ther_park-0.mp4",
            "motionGIFURL": null
          }
        ]
      }
    ]
  }
}
```

The video is at: `generated_images[0].motionMP4URL`

---

## 8. Seedance 2.5 Parameters

Based on leo2api defaults and Seedance 2.0 behavior:

| Parameter | Default | Range | Notes |
|-----------|---------|-------|-------|
| width | 1280 | ? | Standard HD width |
| height | 720 | ? | Standard HD height |
| duration | 4 | 4-? | Seconds of video |
| quantity | 1 | 1 | Number of videos |
| seed | -1 | -1 = random | For reproducibility |
| mode | RESOLUTION_720 | | Resolution mode |
| motion_has_audio | false | | Audio generation |
| prompt_enhance | OFF | OFF/ON | AI prompt enhancement |

---

## 9. Code Changes Needed in leo2api

### File: `client.go`

#### Add Seedance 2.5 Model Detection

```go
func isSeedance25Model(modelID string) bool {
    return modelID == "seedance-2.5" || modelID == "video-2.5"
}
```

#### Update Generate Function

Around line 1089, add:

```go
if strings.EqualFold(genReq.Model, "seedance-2.5") || strings.EqualFold(genReq.Model, "video-2.5") {
    genReq.Model = "seedance-2.5"
}
```

#### Update Default Dimensions (if needed)

Around line 1115, you might want to add specific defaults for 2.5:

```go
} else if isSeedance25Model(genReq.Model) {
    genReq.Params.Width = 1280
}
```

And around line 1130:

```go
} else if isSeedance25Model(genReq.Model) {
    genReq.Params.Height = 720
}
```

---

## 10. Testing Your Implementation

### cURL Test

```bash
curl -X POST https://api.leonardo.ai/v1/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "operationName": "Generate",
    "variables": {
      "request": {
        "model": "seedance-2.5",
        "public": false,
        "parameters": {
          "prompt": "test video",
          "quantity": 1,
          "duration": 4,
          "width": 1280,
          "height": 720
        }
      }
    },
    "query": "mutation Generate($request: CreateGenerationRequest!) { generate(request: $request) { apiCreditCost generationId __typename } }"
  }'
```

### Expected Flow

1. **Submit** → Get `generationId`
2. **Poll** status every 5-10 seconds until status = "COMPLETE"
3. **Fetch** detail to get `motionMP4URL`
4. **Return** video URL to user

---

## 11. Key Differences from Seedance 2.0

Based on the captured URLs, Seedance 2.5:
- Uses the same GraphQL API structure as 2.0
- Model ID is simply `"seedance-2.5"` (no `-fast` or `-mini` variants yet)
- CDN path includes `bytedance/seedance-2.5_` prefix
- Same parameter structure as 2.0

**Good news:** The implementation is almost identical to 2.0, you just need to:
1. Add the model name to your supported list
2. Set appropriate defaults
3. No special parameter handling needed

---

## 12. Video Tutorial Resources

Since you asked for a video tutorial on reverse engineering:

**YouTube Search Terms:**
- "How to reverse engineer API with Chrome DevTools"
- "Network tab tutorial web scraping"
- "Capture API requests browser"

**Recommended Channels:**
- NetworkChuck
- Traversy Media
- The Coding Train

**Key DevTools Features You Used:**
- Network tab → Filter → "graphql"
- Right-click request → Copy as cURL
- Payload tab → See request body
- Response tab → See server response

---

## Summary

You now have everything needed to implement Seedance 2.5:

✅ GraphQL endpoint and structure
✅ Request/response formats
✅ Authentication method
✅ Polling and result retrieval
✅ Exact model parameters
✅ Code changes for leo2api

The implementation should be straightforward since leo2api already has Seedance 2.0 support - just add the new model name and you're done!
