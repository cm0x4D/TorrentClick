#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#include <dlfcn.h>
#import "icon_darwin.h"

void setDarwinIconDisplay(int showDock, int showTray) {
    dispatch_async(dispatch_get_main_queue(), ^{
        // 1. Activation policy (macOS Dock Icon)
        if (NSApp) {
            NSApplicationActivationPolicy targetPolicy = showDock ?
                NSApplicationActivationPolicyRegular :
                NSApplicationActivationPolicyAccessory;

            if ([NSApp activationPolicy] != targetPolicy) {
                [NSApp setActivationPolicy:targetPolicy];
            }
        }

        // 2. Status item (macOS Menu Bar / System Tray Icon)
        void (^updateStatusItem)(void) = ^{
            Class cls = objc_getClass("SystrayAppDelegate");
            if (!cls) return;
            Ivar ivar = class_getInstanceVariable(cls, "statusItem");
            if (!ivar) return;

            // Dynamically resolve owner symbol without compile-time linkage
            id owner = nil;
            void *ownerSym = dlsym(RTLD_DEFAULT, "owner");
            if (ownerSym) {
                owner = *(id *)ownerSym;
            }

            NSStatusItem *item = nil;
            if (owner && [owner isKindOfClass:cls]) {
                item = (NSStatusItem *)object_getIvar(owner, ivar);
            }
            if (!item && [NSApp delegate] && [[NSApp delegate] isKindOfClass:cls]) {
                item = (NSStatusItem *)object_getIvar([NSApp delegate], ivar);
            }
            if (item) {
                item.visible = showTray ? YES : NO;
            }
        };

        updateStatusItem();
        // Also schedule a secondary update after 350ms in case the systray was still initializing
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(350 * NSEC_PER_MSEC)), dispatch_get_main_queue(), updateStatusItem);
    });
}
