package verifier

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneImportFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tf-prune-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create dummy terraform files
	mainTF := filepath.Join(tempDir, "main.tf")
	varsTF := filepath.Join(tempDir, "variables.tf")
	netImports := filepath.Join(tempDir, "networking_imports.tf")
	computeImports := filepath.Join(tempDir, "compute_imports.tf")

	if err := os.WriteFile(mainTF, []byte("resource \"google_compute_network\" \"vpc\" {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(varsTF, []byte("variable \"project_id\" {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(netImports, []byte("import { to = module.vpc id = \"vpc-id\" }"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(computeImports, []byte("import { to = module.compute id = \"vm-id\" }"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run Prune with backup enabled
	pruned, err := PruneImportFiles(tempDir, true)
	if err != nil {
		t.Fatalf("PruneImportFiles failed: %v", err)
	}

	if pruned != 2 {
		t.Errorf("expected 2 pruned files, got %d", pruned)
	}

	// Verify main.tf and variables.tf still exist
	if _, err := os.Stat(mainTF); os.IsNotExist(err) {
		t.Errorf("main.tf was unexpectedly deleted")
	}
	if _, err := os.Stat(varsTF); os.IsNotExist(err) {
		t.Errorf("variables.tf was unexpectedly deleted")
	}

	// Verify import files are removed from root
	if _, err := os.Stat(netImports); !os.IsNotExist(err) {
		t.Errorf("networking_imports.tf was not deleted from root")
	}
	if _, err := os.Stat(computeImports); !os.IsNotExist(err) {
		t.Errorf("compute_imports.tf was not deleted from root")
	}

	// Verify backup exists in .archive/imports/
	archivedNet := filepath.Join(tempDir, ".archive", "imports", "networking_imports.tf")
	if _, err := os.Stat(archivedNet); os.IsNotExist(err) {
		t.Errorf("archived networking_imports.tf not found in .archive/imports")
	}
}
