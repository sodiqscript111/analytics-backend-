terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">= 3.0.0"
    }
  }
}

provider "azurerm" {
  features {}

}

# Random suffix for uniqueness
resource "random_string" "suffix" {
  length  = 6
  upper   = false
  special = false
}

resource "azurerm_resource_group" "analytics_rg" {
  location = "centralus"
  name     = "analytics_resource_group"
}

resource "azurerm_postgresql_flexible_server" "analytics_server" {
  location            = azurerm_resource_group.analytics_rg.location
  name                = "analytics-${random_string.suffix.result}"
  resource_group_name = azurerm_resource_group.analytics_rg.name
  version             = "12"
  administrator_login = "analyticsadmin"
  administrator_password = "Analytics@130####"
    storage_mb         = 32768
    sku_name           = "GP_Standard_D4s_v3"
}

resource "azurerm_postgresql_flexible_server_configuration" "analytics_config" {
  name      = "log_checkpoints"
  server_id = azurerm_postgresql_flexible_server.analytics_server.id
  value     = "on"
}

resource "azurerm_postgresql_flexible_server_database" "analytics_db" {
  collation = "en_US.utf8"
  charset   = "UTF8"
  name      = "analyticsdb"
  server_id = azurerm_postgresql_flexible_server.analytics_server.id
  lifecycle {
    prevent_destroy = true
  }
}

resource "azurerm_postgresql_flexible_server_firewall_rule" "allowAll" {
  name             = "allowall"
  server_id        = azurerm_postgresql_flexible_server.analytics_server.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "255.255.255.255"
}

