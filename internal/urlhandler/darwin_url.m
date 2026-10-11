#import <Cocoa/Cocoa.h>
#import "darwin_url.h"

// Forward declaration of Go export
extern void handleDarwinURL(char* urlStr);

@interface TorrentURLHandler : NSObject
- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent;
@end

@implementation TorrentURLHandler
- (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    NSString *url = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    if (url) {
        handleDarwinURL((char *)[url UTF8String]);
    }
}
@end

static TorrentURLHandler *handlerInstance = nil;

void setupDarwinURLHandler(void) {
    if (handlerInstance == nil) {
        handlerInstance = [[TorrentURLHandler alloc] init];
        NSAppleEventManager *em = [NSAppleEventManager sharedAppleEventManager];
        [em setEventHandler:handlerInstance
                andSelector:@selector(handleGetURLEvent:withReplyEvent:)
              forEventClass:kInternetEventClass
                 andEventID:kAEGetURL];
    }
}
