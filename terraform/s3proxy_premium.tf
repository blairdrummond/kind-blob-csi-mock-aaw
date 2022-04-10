resource "azurerm_storage_account" "premium_s3proxy" {
  name                     = "blobcsidriverp"
  resource_group_name      = var.resource_group
  location                 = var.location
  account_tier             = "Standard"
  account_replication_type = "ZRS"

  account_kind             = "StorageV2"
  # Use object versioning instead
  is_hns_enabled           = "false"

  depends_on = [azurerm_resource_group.s3proxy]
}
