# The path follows a pattern
# ./dist/BUILD-ID_TARGET/BINARY-NAME
source = ["./dist/space-darwin_darwin_amd64/space", "./dist/space-darwin_darwin_386/space"]
bundle_id = "co.fleek.space"

apple_id {
  username = "daniel@fleek.co"
  password = "@env:APPLE_DEVELOPER_DANIEL_PASSWORD"
}

sign {
  application_identity = "Mac Developer: Daniel Merrill (8257VLCFL7)"
}