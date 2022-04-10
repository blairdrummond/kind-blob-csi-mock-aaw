output "name" {
  value       = azurerm_storage_account.s3proxy.name
  description = "Storage account name."
}

output "access_key" {
  value       = azurerm_storage_account.s3proxy.secondary_access_key
  description = "Storage account access key."
  sensitive   = true
}

# Premium
output "premium_name" {
  value       = azurerm_storage_account.premium_s3proxy.name
  description = "Storage account name."
}

output "premium_access_key" {
  value       = azurerm_storage_account.premium_s3proxy.secondary_access_key
  description = "Storage account access key."
  sensitive   = true
}

# Fdi_Prob
output "fdi_prob_name" {
  value       = azurerm_storage_account.fdi_prob_s3proxy.name
  description = "Storage account name."
}

output "fdi_prob_access_key" {
  value       = azurerm_storage_account.fdi_prob_s3proxy.secondary_access_key
  description = "Storage account access key."
  sensitive   = true
}

# Fdi_Unclassified
output "fdi_unclassified_name" {
  value       = azurerm_storage_account.fdi_unclassified_s3proxy.name
  description = "Storage account name."
}

output "fdi_unclassified_access_key" {
  value       = azurerm_storage_account.fdi_unclassified_s3proxy.secondary_access_key
  description = "Storage account access key."
  sensitive   = true
}
