terraform {
  required_providers {
    bitrise = {
      source = "hashicorp.com/edu/bitrise"
    }
  }
}

provider "bitrise" {
  token    = "9a7MN1mZQFMa5DQp1k5szrG17rGZmIx67hoXRVjcfOQBIJ0HX50AN6iungPgIwMMMx3Iq9mKL8o4o8z6B_lvcA"
  endpoint = "https://api.bitrise.io"
}

# resource "bitrise_app" "app_resource" {
#   repo                = "github"
#   is_public           = false
#   organization_slug   = "1fb2d79245bb77e9"
#   repo_url            = "git@github.com:dmalicia/fictional-winner.git"
#   type                = "git"
#   git_repo_slug       = "example-repository"
#   git_owner           = "api_demo"
#     # Capture the app_slug in the state file or variable
#   lifecycle {
#     create_before_destroy = true
#     ignore_changes        = [app_slug]
#   }
# }

# resource "bitrise_app_finish" "app_name_finish" {
#   app_slug          = bitrise_app.app_resource.app_slug
#   project_type      = "ios"
#   stack_id          = "osx-xcode-13.2.x"
#   config            = "default-ios-config"
#   mode              = "manual"
#   envs = {
#     env1 = "val1"
#     env2 = "val2"
#   }
#   organization_slug  = "1fb2d79245bb77e9"
#   depends_on = [bitrise_app_ssh.ssh_resource]
# }

# resource "bitrise_app_ssh" "ssh_resource" {
#   app_slug                            = bitrise_app.app_resource.app_slug
#   auth_ssh_private_key                = <<EOT
# -----BEGIN RSA PRIVATE KEY-----
# MIIJKAIBAAKCAgEAwnXtpxZp3ClkRnDK8mnMRxajnIbGxPD6QOpGHeowK4CLjDpX
# 2JloJim1nEDOEYypQWGUjEncN97JKI48XHT+pXsvU5GkTzl/8NSVcLt+MX3/tMgO
# sQt/uDYKEbi0BjCYAoW6qI27wcexAwvL5XeQtkhRRYZ5q8HEMSUAnQQZkB8i6Pdn
# 231Y6CQ/sJFOAagz1WZaIxmnH2ArrzD7GtscN/dvV9BTOBE1/omftNXUuj1gpLbS
# 5pCndlLLk8XeRrAqlBjQKwhB12RG8eGTYzzElp+ltDWaDETW708wyxEle2wch7A6
# Y0akZAE/LK9Xyc9HzdaiHwWorbFZURVibLglxzJt/3CsJMlcLI9P65F0VLvIhS00
# exxEkKjatxaaxJiFSx7wdbRMr3tmef3ayfTuzzL5QiGvuz0hk3lpENTACfgSax/c
# d3NAWO4k2vFMa7dSdb7O6kZQyJAIPkLIuDKevEotugRUDpqvGcEBXUrN2MxawyV6
# nBJ4b8xguiyW8gBIO8HhzGS1TSqwhCtUAU0oHX/vWF9RQsL4cuTE9hB1PPolJiTl
# LjaNHxJeqilE0ZgYa9GIQfauoysacN8a8q7CXHJzquamWBmr8jpY++Wkk5/fOD54
# GW+o78Qs1BULFadAv7UNMSIYVRUXw0bv7/WwHw6u5+bq9QEcyFcHd7frKVcCAwEA
# AQKCAgEAoipwJqAVZcmK2wdi52d9OGdTx8vJZSFEwO/dy0KqKw0G0skwyuubo/+y
# ePy+HHp+B40VsSxDHsCGZnC/O6dBWMTywbE6Ietkm3TcrudcpG9b1+nh/pkFSJyg
# Jwkt79+EVM4qzDduNXqPTmf/AHyGTMzgIae0PZzYPNeLvGVX4A1nMnpnvO26P9VC
# 279BGzanCzZQwua4rPypUW76aPoCfVW2H7gWPjJ0IbGpYsfToABhYNsp46cMUCtZ
# pAEljTOKPni22LwJFFOGql4gaGib1LSMHk15CvQ5fdY/bYj+BgAxhqJa/sFBhDhZ
# 86zB/AUE704nYtF4SmkUf/7iVqH2tcQ1VTnUgE/KS4IWqAMPKL9aM38v8o+ZyM5S
# BoHF4rPae+mJ9z26W8K02x02jHDyAO7ZHbPGUTNMWoQ0nu/qQELtezwlk458iqNB
# A0/Gx2cTpZyGKZKs/7qTAolH5ZOwcTJiSDdhwSfMIFYYtzFBPxh8LLSZzGwzj+My
# lZj2R4wgnXvajw+1lnTbjxIoA40y1CcFWq2u2QGgLX5aL1VV+6cd3u6HoxfdxhJx
# TRpLRHoaUwiyJHTwFvhrI6pDlXLJJnPqPrugFf1yi7raNP6fVQvLuXuxCONFWS2J
# wueux5nrLxOgR7i0MrJzNOXk2EcarzcK4gT+d7Kk0FpxP0Zo2UECggEBAOX5G1+n
# kb5Z+yT55G4ydqoNAKSngIFTS33FMT6G4kIQNJyH4pfS+yIw+pjNMBUeiiNBhRTp
# MK54CNPrS4dF7FTReBoqTuniugOEWY7ad0Z4qI7q01Nw4p5VxNYvr2ln5ftQHztm
# MP8C0AEwIwz41BTExLGur+RIkrrYpSDLgGmxqiceHZ1u+7WAGZS77ZrpOUsvMQN2
# GU2ajAKjDFOjyrBwvXZrkLnPMj38ilwBcAX2coiZWH/HJKA9TqzwY6fDNzGkLXzv
# dKWDBLroUfyMjhzeAlPfdeQ22ajpAewNeh5nNim3Ew9X38QYCiy+bE1eb9XPjKe6
# G1m4QHo2xWlnfWECggEBANh38C3FJ91jJdRuGopkFfBUlw1w77i1UjohU9s21Hgm
# Mw8SkPekO9q8RiZqDX+4zZIdCME69vCSRZh1oQuTV+9mTK6rxBTEyGLC/LHFGMRF
# CAXyKQQcAeHQioBGDmKreygF2i5YPmI3buqjmmDKypkdVdvQYIYnhoMdMl2v9uLq
# LoY5c2H/K7AT2S+SVLcZ7qgwFOhxMLw+5ud+GixjHzcIDXiNwjG2b38HJanb63ul
# MYWnAqaGYk8ag3uXzqi4waemuwOdsJnGfZaSCeNz4NR4vPT84pwnGLYd5q3XrwuV
# fJpe5NMEAXnCqNayX33nncLYyMtemeq1zMFMSt1cKbcCggEAHXHKnnGvCGcu76oL
# JEzTwqwNhAdqPaSzirPfvTi56Wl3wv7m9TdvLg6FV2EWIe4aE6+E4YuFzyDRSIjc
# z8IVIzr6nKcEGZAM7vxYFyFDmkNCmaHZUtqmOU2T+TR7ygwidw3oIcvQxCXRCgXm
# xvdo+AvFf1Z1cM4V81RfuBY2J9I2jfGeKxUVp0RyggeZwXbQ/h5ZsS7CyJvcB05m
# +qKDBho5N9tH2XJ85VDbSjJo7GqEeZbgrOOLffS7iQryR32IKJPzuwZRsgtXZLLw
# JFy+qVWHiMOYrZKURbsnotiK5S+j0K1/BDzlCo9lZhsvdKx9tytuv97lN5SOtNck
# aX11oQKCAQAQpNArxL/27rum5LxXrbBeJTLkDq3v5skmvQ9EiKe3gUBlxUiuMcuE
# WvuN0pOtIVl1BZR5vv3jq2t8eHbke/TD7Hqy53QRILxDk7h4Nq5b0O73/hGXRkwC
# v9UWXcyXW5YBksmezJwnUxnNIr0o+g6vzSif2RrC1eEqzaDkwTXbZqQjH+G2RDdo
# t234kWjAF1dZSTEiWimkH7YDUJfUl957jbvza/rldaCHBNapg8ZMYHw5SYkToruG
# V4SKiTaTlHkXWeOBOKuudyuK2zm1amB8Fbh5ocQOu5bT0eK9tRq5akoFWIyBiQpQ
# AV6X+2kKNjfUFnUB5gkxmb7fke0jrgVrAoIBAB9mYJ9SjgrWfxn0LSjR7BtkVgHt
# hfX8pRjIcapPJmSCWMqkYFrH5qXj47kS8C1mh7c06WMrs7VCtpmnN1AQ0hy7nXFV
# fvQy0IN79/bWb8oZu97YOEXalDpq121vyO+n5WE/32Y0LiRSiOVziN+sQhMSEDq9
# 8pVZ2IPeYxc/nOiaE0og6EjvU+TuNJAn0Vn788/STSOn1owaT434dwJIeO7/zraI
# FqiGjaO9pkuJszpY9IuaG4iADfDVJXvaw8JKN45q5hoCmqXoYPztFrAx1D7Wog5a
# niLq2jMc/487vAuZJTcQ8mmnjvW+yOfJaoMu7jrRzOZVBAn6/fy4GuwnmUE=
# -----END RSA PRIVATE KEY-----
# EOT
#   auth_ssh_public_key                 = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDCde2nFmncKWRGcMryacxHFqOchsbE8PpA6kYd6jArgIuMOlfYmWgmKbWcQM4RjKlBYZSMSdw33skojjxcdP6ley9TkaRPOX/w1JVwu34xff+0yA6xC3+4NgoRuLQGMJgChbqojbvBx7EDC8vld5C2SFFFhnmrwcQxJQCdBBmQHyLo92fbfVjoJD+wkU4BqDPVZlojGacfYCuvMPsa2xw3929X0FM4ETX+iZ+01dS6PWCkttLmkKd2UsuTxd5GsCqUGNArCEHXZEbx4ZNjPMSWn6W0NZoMRNbvTzDLESV7bByHsDpjRqRkAT8sr1fJz0fN1qIfBaitsVlRFWJsuCXHMm3/cKwkyVwsj0/rkXRUu8iFLTR7HESQqNq3FprEmIVLHvB1tEyve2Z5/drJ9O7PMvlCIa+7PSGTeWkQ1MAJ+BJrH9x3c0BY7iTa8Uxrt1J1vs7qRlDIkAg+Qsi4Mp68Si26BFQOmq8ZwQFdSs3YzFrDJXqcEnhvzGC6LJbyAEg7weHMZLVNKrCEK1QBTSgdf+9YX1FCwvhy5MT2EHU8+iUmJOUuNo0fEl6qKUTRmBhr0YhB9q6jKxpw3xryrsJccnOq5qZYGavyOlj75aSTn984PngZb6jvxCzUFQsVp0C/tQ0xIhhVFRfDRu/v9bAfDq7n5ur1ARzIVwd3t+spVw== dmalicia@Diegos-Air"
#   is_register_key_into_provider_service = false
#   depends_on = [bitrise_app.app_resource]
# }






