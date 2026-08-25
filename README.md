# Seedance 2.5 - Complete Reverse Engineering Documentation

This repository contains complete documentation and implementation guide for adding **Seedance 2.5** support to Leonardo.ai API clients.

## 📋 What's Inside

- **SEEDANCE_2.5_IMPLEMENTATION_GUIDE.md** - Complete API reference with GraphQL queries, parameters, and step-by-step implementation
- **seedance25_example.go** - Working Go code example (standalone implementation)
- **SEEDANCE_2.5_SUMMARY.txt** - Quick reference guide
- **leonardo_seedance_2.5_capture.jsonl** - Raw network capture data (2,835 events)
- **leonardo_seedance_2.5_summary.txt** - API endpoints summary
- **capture_leonardo_seedance.py** - Reusable network capture script
- **analyze_leonardo_capture.py** - Analysis script for captured data
- **manual_capture_guide.sh** - Manual capture instructions using Chrome DevTools

## 🚀 Quick Start

### For leo2api Users

If you're using [leo2api](https://github.com/example/leo2api), you only need **4 small code additions**:

```go
// 1. Add model detection function
func isSeedance25Model(modelID string) bool {
    return modelID == "seedance-2.5" || modelID == "video-2.5"
}

// 2. Add model alias
if strings.EqualFold(genReq.Model, "seedance-2.5") {
    genReq.Model = "seedance-2.5"
}

// 3. Set default width
} else if isSeedance25Model(genReq.Model) {
    genReq.Params.Width = 1280
}

// 4. Set default height
} else if isSeedance25Model(genReq.Model) {
    genReq.Params.Height = 720
}
```

See `SEEDANCE_2.5_IMPLEMENTATION_GUIDE.md` for exact line numbers and complete details.

### For Custom Implementation

Check `seedance25_example.go` for a complete standalone implementation in Go.

## 📊 Key Findings

- **API Endpoint:** `POST https://api.leonardo.ai/v1/graphql`
- **Model Name:** `seedance-2.5`
- **Provider:** ByteDance
- **Default Resolution:** 1280x720 (16:9)
- **Default Duration:** 4 seconds
- **Same API structure as Seedance 2.0** ✅

## 🔍 How This Was Reverse Engineered

1. Captured live network traffic from Leonardo.ai using Playwright
2. Analyzed 2,835 API events during video generation
3. Identified GraphQL mutations, queries, and response structures
4. Validated against leo2api source code
5. Documented complete request/response flow

## 📖 Documentation

- Start with `SEEDANCE_2.5_SUMMARY.txt` for a quick overview
- Read `SEEDANCE_2.5_IMPLEMENTATION_GUIDE.md` for complete details
- Study `seedance25_example.go` for working code

## 🎓 Learning Resources

Want to learn reverse engineering? This project demonstrates:
- Network traffic capture with Playwright
- GraphQL API analysis
- Browser DevTools usage
- Reading and extending existing codebases

Search YouTube for:
- "Chrome DevTools Network Tab Tutorial"
- "How to Reverse Engineer APIs"
- "GraphQL API Reverse Engineering"

## 🤝 Contributing

Found improvements or additional parameters? Pull requests welcome!

## 📄 License

MIT

## ⚠️ Disclaimer

This documentation is for educational purposes. Make sure you comply with Leonardo.ai's Terms of Service when using their API.

---

**Status:** ✅ Complete and tested  
**Difficulty:** ⭐⭐☆☆☆ (Very Easy - if using leo2api)  
**Last Updated:** August 2026
