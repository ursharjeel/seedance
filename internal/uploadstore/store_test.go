package uploadstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreSaveLoadMeta(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	meta := Meta{ID: "c66703d3-24d5-4078-b177-3757f5af7cdd", Kind: KindImage, Ext: "jpg", ContentType: "image/jpeg", OriginalFilename: "download.jpg"}
	if err := store.Save(meta, []byte("fake-jpeg-bytes")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	gotMeta, ok := store.Meta(meta.ID)
	if !ok || gotMeta.Kind != KindImage || gotMeta.ContentType != "image/jpeg" || gotMeta.Size != int64(len("fake-jpeg-bytes")) {
		t.Fatalf("Meta = %+v, ok=%v", gotMeta, ok)
	}
	data, loadedMeta, ok := store.Load(meta.ID)
	if !ok || string(data) != "fake-jpeg-bytes" {
		t.Fatalf("Load data mismatch: ok=%v data=%q", ok, data)
	}
	if loadedMeta.ID != meta.ID {
		t.Fatalf("Load meta id = %q", loadedMeta.ID)
	}
	if store.Count() != 1 {
		t.Fatalf("Count = %d, want 1", store.Count())
	}
}

func TestStorePersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	meta := Meta{ID: "14efb56c-f697-40c3-a6cb-261001dded67", Kind: KindAudio, Ext: "mp3", ContentType: "audio/mpeg", OriginalFilename: "speech.mp3"}
	if err := store.Save(meta, []byte("fake-mp3-bytes")); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	data, loadedMeta, ok := reopened.Load(meta.ID)
	if !ok || string(data) != "fake-mp3-bytes" || loadedMeta.Kind != KindAudio {
		t.Fatalf("reloaded = ok:%v data:%q meta:%+v", ok, data, loadedMeta)
	}
	if reopened.Count() != 1 {
		t.Fatalf("reopened Count = %d, want 1", reopened.Count())
	}
}

func TestStoreMissingLookup(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := store.Meta("does-not-exist"); ok {
		t.Fatal("Meta unexpectedly found")
	}
	if _, _, ok := store.Load("does-not-exist"); ok {
		t.Fatal("Load unexpectedly found")
	}
	if _, _, ok := (*Store)(nil).Load("x"); ok {
		t.Fatal("nil store Load unexpectedly found")
	}
}

func TestStoreCorruptMetaIgnored(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	meta := Meta{ID: "abc", Kind: KindAudio}
	if err := store.Save(meta, []byte("data")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	sum := objectKey("abc")
	if err := os.WriteFile(filepath.Join(dir, sum+".meta.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("corrupt write: %v", err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if reopened.Count() != 0 {
		t.Fatalf("Count = %d, want 0 after corrupt meta", reopened.Count())
	}
}
