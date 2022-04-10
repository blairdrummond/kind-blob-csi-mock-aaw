resource "azurerm_resource_group" "s3proxy" {
  name     = var.resource_group
  location = var.location
}
