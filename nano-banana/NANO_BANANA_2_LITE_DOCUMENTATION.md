# Nano Banana 2 Lite - Complete API Documentation

**Captured:** 2026-08-27  
**Status:** ✅ Verified end-to-end  
**Evidence:** Real GraphQL capture from Leonardo.ai

---

## ⚠️ Important Discovery

**Nano Banana 2 Lite is an IMAGE model, NOT a video model!**

Despite being in the video generation section, it generates static images, not videos.

---

## API Endpoint

```
POST https://api.leonardo.ai/v1/graphql
Content-Type: application/json
Authorization: Bearer <JWT_TOKEN>
```

---

## Complete Request

### GraphQL Mutation

```graphql
mutation Generate($request: CreateGenerationRequest!) {
  generate(request: $request) {
    apiCreditCost
    generationId
    cost {
      amount
      unit
      __typename
    }
    __typename
  }
}
```

### Variables

```json
{
  "request": {
    "model": "nano-banana-2-lite",
    "public": true,
    "parameters": {
      "height": 1264,
      "width": 848,
      "prompt_enhance": "OFF",
      "quantity": 1,
      "style_ids": [
        "111dc692-d470-4eec-b791-3475abac4c46"
      ],
      "prompt": "generate galaxy tree in middle of world where millions space ship traffic everywhere",
      "guidances": {
        "image_reference": [
          {
            "image": {
              "id": "2dd7c3f2-96a5-4732-8737-7137bf6b47a5",
              "type": "UPLOADED"
            }
          },
          {
            "image": {
              "id": "0cdaebea-b432-42b2-ba5a-7e6fd7fccc3c",
              "type": "UPLOADED"
            }
          }
        ]
      }
    }
  }
}
```

---

## Parameters

### Required Parameters
- `model`: `"nano-banana-2-lite"`
- `parameters.prompt`: Text prompt (string)
- `parameters.width`: Image width (integer)
- `parameters.height`: Image height (integer)
- `parameters.quantity`: Number of images (integer, default: 1)

### Optional Parameters
- `public`: Make generation public (boolean, default: false)
- `parameters.prompt_enhance`: `"ON"` or `"OFF"` (default: OFF)
- `parameters.style_ids`: Array of style IDs (strings)
- `parameters.guidances.image_reference`: Array of reference images

### Image Reference Structure
```json
{
  "image": {
    "id": "uuid-here",
    "type": "UPLOADED"
  }
}
```

### NOT Supported (Unlike Seedance 2.5)
- ❌ `duration` - No video duration
- ❌ `motion_has_audio` - No audio
- ❌ `mode` - No motion mode
- ❌ `seed` - No seed control
- ❌ `video_reference_base` - No video references
- ❌ `audio_reference` - No audio references

---

## Response

### Immediate Response
```json
{
  "data": {
    "generate": {
      "apiCreditCost": null,
      "generationId": "1f1a1fa5-d97a-6030-9a51-b75f1778b3ba",
      "cost": {
        "amount": "35",
        "unit": "CREDITS",
        "__typename": "Cost"
      },
      "__typename": "CreateGenerationResponse"
    }
  }
}
```

**Cost:** 35 CREDITS

---

## Polling Status

Same as Seedance 2.5 - use `GetAIGenerationFeedStatuses` query.

```graphql
query GetAIGenerationFeedStatuses($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
  }
}
```

Variables:
```json
{
  "where": {
    "id": {"_in": ["1f1a1fa5-d97a-6030-9a51-b75f1778b3ba"]},
    "status": {"_in": ["PENDING", "COMPLETE", "FAILED"]}
  }
}
```

Status flow: `PENDING` → `COMPLETE` (or `FAILED`)

---

## Final Result

### Query for Complete Generation

```graphql
query GetGenerationDetail($where: generations_bool_exp = {}) {
  generations(where: $where) {
    id
    status
    createdAt
    generated_images {
      id
      url
      motionModel
      motionMP4URL
    }
  }
}
```

### Response (COMPLETE)

```json
{
  "id": "1f1a1fa5-d97a-6030-9a51-b75f1778b3ba",
  "status": "COMPLETE",
  "createdAt": "2026-08-27T09:33:31.838",
  "generated_images": [
    {
      "id": "eb06413c-d599-4062-8a39-6e413a0af350",
      "url": "https://cdn.leonardo.ai/users/.../nano-banana-2-lite_....jpg",
      "motionModel": null,
      "motionMP4URL": null
    }
  ]
}
```

**Output:** Static JPG image  
**No video generated:** `motionMP4URL` is `null`

---

## Key Differences: Nano Banana vs Seedance 2.5

| Feature | Nano Banana 2 Lite | Seedance 2.5 |
|---------|-------------------|--------------|
| Output Type | **Static Image** | **Video** |
| Cost | 35 credits | 1,168 credits (33x more) |
| Duration | N/A | 4-29+ seconds |
| Audio | ❌ Not supported | ✅ Supported |
| Video References | ❌ Not supported | ✅ Supported |
| Image References | ✅ Multiple | ✅ Supported |
| Prompt Enhance | ✅ ON/OFF | ✅ Supported |
| Style IDs | ✅ Supported | Unknown |
| Use Case | Image generation | Video generation |

---

## Use Cases

✅ **Good for:**
- Static image generation
- Image-to-image with references
- Low-cost generations (35 credits)
- Portrait/landscape images
- Style-based generation

❌ **NOT for:**
- Video generation
- Animated content
- Audio-driven generation

---

## Upload Workflow

Same as Seedance 2.5 - use `UploadImage` mutation to get upload ID, then use in guidances.

See `REFERENCE_WORKFLOWS.md` in the `sd2.5` folder for complete upload documentation.

---

## Example: Minimal Python Implementation

```python
import requests
import json
import time

JWT_TOKEN = "your_token_here"
API_URL = "https://api.leonardo.ai/v1/graphql"

def generate_nano_banana(prompt, width=848, height=1264):
    """Generate image with Nano Banana 2 Lite"""
    
    mutation = """
    mutation Generate($request: CreateGenerationRequest!) {
      generate(request: $request) {
        generationId
        cost { amount unit }
      }
    }
    """
    
    variables = {
        "request": {
            "model": "nano-banana-2-lite",
            "public": False,
            "parameters": {
                "prompt": prompt,
                "width": width,
                "height": height,
                "quantity": 1,
                "prompt_enhance": "OFF"
            }
        }
    }
    
    response = requests.post(
        API_URL,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {JWT_TOKEN}"
        },
        json={
            "operationName": "Generate",
            "variables": variables,
            "query": mutation
        }
    )
    
    data = response.json()['data']['generate']
    print(f"Generation ID: {data['generationId']}")
    print(f"Cost: {data['cost']['amount']} credits")
    
    return data['generationId']

# Use it
gen_id = generate_nano_banana("a beautiful galaxy tree")
```

---

## Tested Configuration

✅ **Model:** `nano-banana-2-lite`  
✅ **Resolution:** 848x1264 (portrait)  
✅ **Prompt Enhancement:** OFF  
✅ **Image References:** 6 images simultaneously  
✅ **Style IDs:** 1 style  
✅ **Cost:** 35 CREDITS  
✅ **Output:** JPG image  
✅ **Status Flow:** PENDING → COMPLETE  

---

## What's Still Unknown

- Supported resolution ranges
- Maximum prompt length
- Maximum reference images
- Available style IDs
- Seed parameter support
- Image-to-image strength control
- Generation time estimates
- Error responses

---

## Production Ready?

✅ **YES** for basic image generation  
⚠️ **NO** for video generation (wrong model)

The API is fully functional for generating static images with Nano Banana 2 Lite, but if you need video generation, use Seedance 2.5 instead.
