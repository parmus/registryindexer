# Changelog


## 0.1.0
First release on Github

### Known issues
- Registryindexer will sometimes get access denied when pulling images from Artifact Registry.
  The problem seems to be that Google's tokens sometimes expire early for some reason. The current work-around is to simply restart registryindexer.
- Registryindexer can hit API rate limits when reindexer everything on startup. This happens if you have a lot (10k+) images. Currently, the only solution
  is to lower the number of images.
