#ifndef VPNFOUNDATION_H
#define VPNFOUNDATION_H

#include <stdbool.h>
#include <stdint.h>
#include <sys/types.h>

typedef void (*logger_fn_t)(void *context, int level, const char *msg);
extern void wgSetLogger(void *context, logger_fn_t logger_fn);
extern int wgTurnOn(const char *settings, int32_t tun_fd);
extern void wgTurnOff(int handle);
extern int64_t wgSetConfig(int handle, const char *settings);
extern char *wgGetConfig(int handle);
extern void wgBumpSockets(int handle);
extern void wgDisableSomeRoamingForBrokenMobileSemantics(int handle);
extern const char *wgVersion();

extern char *LibXrayRunXray(const char *datDir, const char *configPath, int64_t maxMemory);
extern char *LibXrayRunXrayFromJSON(const char *datDir, const char *configPath);
extern char *LibXrayStopXray();
extern char *LibXrayXrayVersion();
extern int LibXrayGetXrayState();
extern char *LibXrayTestXray(const char *datDir, const char *configPath);
extern char *LibXrayPing(const char *datDir, const char *configPath, int timeout, const char *url, const char *proxy);
extern char *LibXrayCountGeoData(const char *datDir, const char *name, const char *geoType);
extern char *LibXrayReadGeoFiles(const char *data);

#endif
