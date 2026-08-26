# Two runtimes, because the split is clean and both eras are worth supporting:
# Minecraft 1.21.x requires Java 21, and 26.x requires Java 25. Shipping both
# costs around 35 MB compressed and removes the need to download a runtime at
# boot for the overwhelming majority of manifests.
#
# A manifest asking for anything else gets a runtime fetched at boot. The agent
# picks between what is installed by reading javaVersion from the version
# document, not by trusting the manifest.
openjdk-21-jre
openjdk-25-jre
libopenal1
libglfw3-wayland
