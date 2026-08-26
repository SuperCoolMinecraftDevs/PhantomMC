# Java 21 covers Minecraft 1.20.5 and newer, which is the target for the first
# release. Manifests asking for a different major get a runtime fetched at boot
# rather than a second JRE baked into every image.
openjdk-21-jre
libopenal1
libglfw3-wayland
