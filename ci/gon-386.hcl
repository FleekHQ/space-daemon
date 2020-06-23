# The path follows a pattern
# ./dist/BUILD-ID_TARGET/BINARY-NAME
source = ["./dist/space-darwin_darwin_i386/space"]
bundle_id = "co.fleek.space"

apple_id {
  username = "daniel@fleek.co"
  password = "@env:APPLE_DEVELOPER_DANIEL_PASSWORD"
}

sign {
  application_identity = "Mac Developer: Daniel Merrill (8257VLCFL7)"
}

dmg {
  output_path = "dist/space-macos-i386.dmg"
  volume_name = "Space"
}