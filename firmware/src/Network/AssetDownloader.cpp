#include "AssetDownloader.h"
#include <HTTPClient.h>
#include <FS.h>
#include <SD.h>

bool AssetDownloader::downloadAsset(const String& url, const String& filename) {
    HTTPClient http;
    Serial.printf("[AssetDownloader] Descargando %s de %s\n", filename.c_str(), url.c_str());

    http.begin(url);
    int httpCode = http.GET();

    if (httpCode == HTTP_CODE_OK) {
        File file = SD.open(filename, FILE_WRITE);
        if (!file) {
            Serial.println("[AssetDownloader] Error abriendo archivo para escritura en SD.");
            http.end();
            return false;
        }

        // Escribir el payload a la MicroSD por bloques (stream)
        http.writeToStream(&file);
        file.close();
        Serial.println("[AssetDownloader] Descarga y guardado completado.");
    } else {
        Serial.printf("[AssetDownloader] HTTP GET falló, error: %s\n", http.errorToString(httpCode).c_str());
        http.end();
        return false;
    }

    http.end();
    return true;
}
