#ifndef ASSET_DOWNLOADER_H
#define ASSET_DOWNLOADER_H

#include <Arduino.h>

class AssetDownloader {
public:
    // downloadAsset downloads a binary file from the given URL and saves it to the MicroSD card.
    // Returns true if successful, false otherwise.
    static bool downloadAsset(const String& url, const String& filename);
};

#endif // ASSET_DOWNLOADER_H
