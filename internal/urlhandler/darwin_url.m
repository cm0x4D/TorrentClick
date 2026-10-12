#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>
#import "darwin_url.h"

// Forward declaration of Go export
extern void handleDarwinURL(char* urlStr);

static NSString *extractPathFromDescriptor(NSAppleEventDescriptor *desc) {
    if (!desc) return nil;
    
    // 1. Try fileURLValue (macOS 10.11+)
    if ([desc respondsToSelector:@selector(fileURLValue)]) {
        NSURL *url = [desc fileURLValue];
        if (url && [url path]) {
            return [url path];
        }
    }
    
    // 2. Try typeFileURL
    if ([desc descriptorType] == typeFileURL) {
        NSString *urlStr = [[NSString alloc] initWithData:[desc data] encoding:NSUTF8StringEncoding];
        if (urlStr) {
            NSURL *url = [NSURL URLWithString:urlStr];
            if (url && [url path]) return [url path];
        }
    }
    
    // 3. Try stringValue
    NSString *str = [desc stringValue];
    if (str && [str length] > 0) {
        if ([str hasPrefix:@"file://"]) {
            NSURL *url = [NSURL URLWithString:str];
            if (url && [url path]) return [url path];
        }
        return str;
    }
    
    // 4. Try bookmark data
    if ([desc descriptorType] == typeBookmarkData) {
        BOOL isStale = NO;
        NSError *err = nil;
        NSURL *url = [NSURL URLByResolvingBookmarkData:[desc data]
                                              options:NSURLBookmarkResolutionWithoutUI
                                        relativeToURL:nil
                                  bookmarkDataIsStale:&isStale
                                                error:&err];
        if (url && [url path]) return [url path];
    }
    
    return nil;
}

@interface TorrentURLHandler : NSObject
- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent;
- (void)handleOpenDocsEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent;
@end

@implementation TorrentURLHandler
- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    (void)replyEvent;
    NSString *url = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    if (url) {
        NSLog(@"[TorrentClick] handleGetURLEvent: %@", url);
        handleDarwinURL((char *)[url UTF8String]);
    }
}

- (void)handleOpenDocsEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    (void)replyEvent;
    NSAppleEventDescriptor *docList = [event paramDescriptorForKeyword:keyDirectObject];
    NSInteger count = [docList numberOfItems];
    NSLog(@"[TorrentClick] handleOpenDocsEvent count=%ld", (long)count);
    if (count > 0) {
        for (NSInteger i = 1; i <= count; i++) {
            NSAppleEventDescriptor *itemDesc = [docList descriptorAtIndex:i];
            NSString *path = extractPathFromDescriptor(itemDesc);
            if (path) {
                NSLog(@"[TorrentClick] handleOpenDocsEvent item path: %@", path);
                handleDarwinURL((char *)[path UTF8String]);
            }
        }
    } else {
        NSString *path = extractPathFromDescriptor(docList);
        if (path) {
            NSLog(@"[TorrentClick] handleOpenDocsEvent single path: %@", path);
            handleDarwinURL((char *)[path UTF8String]);
        }
    }
}
@end

static TorrentURLHandler *handlerInstance = nil;

static void registerAppleEventHandlers(void) {
    if (handlerInstance == nil) {
        handlerInstance = [[TorrentURLHandler alloc] init];
    }
    NSAppleEventManager *em = [NSAppleEventManager sharedAppleEventManager];
    [em setEventHandler:handlerInstance
            andSelector:@selector(handleGetURLEvent:withReplyEvent:)
          forEventClass:kInternetEventClass
             andEventID:kAEGetURL];
    [em setEventHandler:handlerInstance
            andSelector:@selector(handleOpenDocsEvent:withReplyEvent:)
          forEventClass:kCoreEventClass
             andEventID:kAEOpenDocuments];
    NSLog(@"[TorrentClick] AppleEvent handlers registered for kAEGetURL and kAEOpenDocuments");
}

static void hookDelegateClass(Class cls) {
    if (!cls) return;
    
    // 1. application:openFiles:
    SEL openFilesSel = @selector(application:openFiles:);
    void (^openFilesBlock)(id, NSApplication *, NSArray<NSString *> *) = ^(id self, NSApplication *app, NSArray<NSString *> *files) {
        (void)self; (void)app;
        NSLog(@"[TorrentClick] delegate application:openFiles: count=%lu", (unsigned long)[files count]);
        for (NSString *file in files) {
            handleDarwinURL((char *)[file UTF8String]);
        }
    };
    IMP openFilesImp = imp_implementationWithBlock(openFilesBlock);
    if (!class_addMethod(cls, openFilesSel, openFilesImp, "v@:@@")) {
        Method origMethod = class_getInstanceMethod(cls, openFilesSel);
        if (origMethod) {
            method_setImplementation(origMethod, openFilesImp);
        }
    }

    // 2. application:openFile:
    SEL openFileSel = @selector(application:openFile:);
    BOOL (^openFileBlock)(id, NSApplication *, NSString *) = ^BOOL(id self, NSApplication *app, NSString *filename) {
        (void)self; (void)app;
        NSLog(@"[TorrentClick] delegate application:openFile: %@", filename);
        if (filename) {
            handleDarwinURL((char *)[filename UTF8String]);
        }
        return YES;
    };
    IMP openFileImp = imp_implementationWithBlock(openFileBlock);
    if (!class_addMethod(cls, openFileSel, openFileImp, "c@:@@")) {
        Method origMethod = class_getInstanceMethod(cls, openFileSel);
        if (origMethod) {
            method_setImplementation(origMethod, openFileImp);
        }
    }
    
    // 3. application:openURLs:
    SEL openURLsSel = @selector(application:openURLs:);
    void (^openURLsBlock)(id, NSApplication *, NSArray<NSURL *> *) = ^(id self, NSApplication *app, NSArray<NSURL *> *urls) {
        (void)self; (void)app;
        NSLog(@"[TorrentClick] delegate application:openURLs: count=%lu", (unsigned long)[urls count]);
        for (NSURL *url in urls) {
            if ([url isFileURL]) {
                handleDarwinURL((char *)[[url path] UTF8String]);
            } else {
                handleDarwinURL((char *)[[url absoluteString] UTF8String]);
            }
        }
    };
    IMP openURLsImp = imp_implementationWithBlock(openURLsBlock);
    if (!class_addMethod(cls, openURLsSel, openURLsImp, "v@:@@")) {
        Method origMethod = class_getInstanceMethod(cls, openURLsSel);
        if (origMethod) {
            method_setImplementation(origMethod, openURLsImp);
        }
    }
    NSLog(@"[TorrentClick] Application delegate hooked for openFile:, openFiles:, and openURLs:");
}

@interface NSApplication (TorrentHook)
- (void)tc_setDelegate:(id<NSApplicationDelegate>)delegate;
@end

@implementation NSApplication (TorrentHook)
- (void)tc_setDelegate:(id<NSApplicationDelegate>)delegate {
    [self tc_setDelegate:delegate];
    if (delegate) {
        hookDelegateClass([delegate class]);
        NSLog(@"[TorrentClick] Hooked delegate class via tc_setDelegate: %@", NSStringFromClass([delegate class]));
    }
}
@end

static void swizzleNSApplicationSetDelegate(void) {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        Class appClass = [NSApplication class];
        SEL origSel = @selector(setDelegate:);
        SEL swizSel = @selector(tc_setDelegate:);
        
        Method origMethod = class_getInstanceMethod(appClass, origSel);
        Method swizMethod = class_getInstanceMethod(appClass, swizSel);
        
        if (origMethod && swizMethod) {
            method_exchangeImplementations(origMethod, swizMethod);
            NSLog(@"[TorrentClick] Swizzled NSApplication setDelegate:");
        }
    });
}

void setupDarwinURLHandler(void) {
    registerAppleEventHandlers();
    swizzleNSApplicationSetDelegate();
    
    if (NSApp && [NSApp delegate]) {
        hookDelegateClass([[NSApp delegate] class]);
    }
    
    NSNotificationCenter *nc = [NSNotificationCenter defaultCenter];
    [nc addObserverForName:NSApplicationWillFinishLaunchingNotification
                    object:nil
                     queue:[NSOperationQueue mainQueue]
                usingBlock:^(NSNotification *note) {
        (void)note;
        registerAppleEventHandlers();
        if (NSApp && [NSApp delegate]) {
            hookDelegateClass([[NSApp delegate] class]);
        }
    }];
    
    [nc addObserverForName:NSApplicationDidFinishLaunchingNotification
                    object:nil
                     queue:[NSOperationQueue mainQueue]
                usingBlock:^(NSNotification *note) {
        (void)note;
        registerAppleEventHandlers();
        if (NSApp && [NSApp delegate]) {
            hookDelegateClass([[NSApp delegate] class]);
        }
    }];
}
