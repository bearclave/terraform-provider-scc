# Managed by bearclave/governance (tools/sync.py) - edit only below the marker.
# The base of tflint in every repository with Terraform: the recommended
# rules of the terraform ruleset and the azurerm ruleset
# (standards/pipelines.md, "Checks and their exceptions"). Deletion
# protection stays on: a resource that does without it says so next to the
# block (# tflint-ignore: azurerm_resources_missing_prevent_destroy # reason).
plugin "terraform" {
  enabled = true
  preset  = "recommended"
}

plugin "azurerm" {
  enabled = true
  version = "0.30.0"
  source  = "github.com/terraform-linters/tflint-ruleset-azurerm"
}

rule "azurerm_resources_missing_prevent_destroy" {
  enabled = true
}

# --- Repository-specific (kept by sync.py) ---
# Rules this repository switches off or adds, each with the reason.
