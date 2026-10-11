#import <Cocoa/Cocoa.h>
#import "dark_darwin.h"

extern void onDarwinThemeChanged(void);

@interface ThemeWatcher : NSObject
- (void)themeChanged:(NSNotification *)note;
@end

@implementation ThemeWatcher
- (void)themeChanged:(NSNotification *)note {
    (void)note;
    onDarwinThemeChanged();
}
@end

static ThemeWatcher *watcherInstance = nil;

int isDarwinDarkMode(void) {
    NSDictionary *globalDomain = [[NSUserDefaults standardUserDefaults] persistentDomainForName:NSGlobalDomain];
    id style = [globalDomain objectForKey:@"AppleInterfaceStyle"];
    if (style && [style isKindOfClass:[NSString class]]) {
        if ([style isEqualToString:@"Dark"]) {
            return 1;
        }
    }

    // Fallback: check NSApp effectiveAppearance if app runloop is active
    if (NSApp) {
        NSAppearanceName appearance = [[NSApp effectiveAppearance] bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
        if ([appearance isEqualToString:NSAppearanceNameDarkAqua]) {
            return 1;
        }
    }

    return 0;
}

void startDarwinThemeObserver(void) {
    if (watcherInstance == nil) {
        watcherInstance = [[ThemeWatcher alloc] init];
        [[NSDistributedNotificationCenter defaultCenter] addObserver:watcherInstance
                                                            selector:@selector(themeChanged:)
                                                                name:@"AppleInterfaceThemeChangedNotification"
                                                              object:nil];
    }
}
