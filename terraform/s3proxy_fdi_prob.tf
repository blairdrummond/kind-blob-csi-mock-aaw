resource "azurerm_storage_account" "fdi_prob_s3proxy" {
  name                     = "blobcsidriverfdipb"
  resource_group_name      = var.resource_group
  location                 = var.location
  account_tier             = "Standard"
  account_replication_type = "ZRS"

  account_kind             = "StorageV2"
  # Use object versioning instead
  is_hns_enabled           = "false"

  depends_on = [azurerm_resource_group.s3proxy]
}

resource "azurerm_storage_container" "fdi_prob_crops" {
  name                  = "crops"
  storage_account_name  = azurerm_storage_account.fdi_prob_s3proxy.name
  container_access_type = "private"

  depends_on = [azurerm_storage_account.fdi_prob_s3proxy]
}


resource "azurerm_storage_container" "fdi_prob_greenhouses" {
  name                  = "greenhouses"
  storage_account_name  = azurerm_storage_account.fdi_prob_s3proxy.name
  container_access_type = "private"

  depends_on = [azurerm_storage_account.fdi_prob_s3proxy]
}
