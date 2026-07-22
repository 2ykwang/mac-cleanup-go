//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>
#import <stdint.h>

static char *copy_response(NSArray<NSString *> *moved_paths, NSString *error_message) {
    NSDictionary *response = @{
        @"moved_paths": moved_paths ?: @[],
        @"error": error_message ?: @""
    };
    NSData *data = [NSJSONSerialization dataWithJSONObject:response options:0 error:nil];
    if (data == nil) {
        return NULL;
    }

    char *output = malloc(data.length + 1);
    if (output == NULL) {
        return NULL;
    }
    memcpy(output, data.bytes, data.length);
    output[data.length] = '\0';
    return output;
}

static NSString *operation_error_message(NSError *error) {
    if (error == nil) {
        return nil;
    }
    return [NSString stringWithFormat:@"NSWorkspace.recycleURLs: %@ (%@/%ld)",
                                      error.localizedDescription,
                                      error.domain,
                                      (long)error.code];
}

@interface MacCleanupRecycleOperation : NSObject {
@private
    dispatch_semaphore_t _completed;
    NSArray<NSString *> *_moved_paths;
    NSError *_operation_error;
}

- (void)completeWithMovedPaths:(NSArray<NSString *> *)moved_paths
                         error:(NSError *)error;
- (BOOL)waitForTimeoutNanoseconds:(int64_t)timeout_nanoseconds;
- (char *)copyResponse;

@end

@implementation MacCleanupRecycleOperation

- (instancetype)init {
    self = [super init];
    if (self != nil) {
        _completed = dispatch_semaphore_create(0);
    }
    return self;
}

- (void)completeWithMovedPaths:(NSArray<NSString *> *)moved_paths
                         error:(NSError *)error {
    _moved_paths = [moved_paths copy];
    _operation_error = [error copy];
    dispatch_semaphore_signal(_completed);
}

- (BOOL)waitForTimeoutNanoseconds:(int64_t)timeout_nanoseconds {
    if (timeout_nanoseconds <= 0) {
        return NO;
    }

    dispatch_time_t deadline = dispatch_time(DISPATCH_TIME_NOW, timeout_nanoseconds);
    return dispatch_semaphore_wait(_completed, deadline) == 0;
}

- (char *)copyResponse {
    return copy_response(_moved_paths, operation_error_message(_operation_error));
}

- (void)dealloc {
    [_moved_paths release];
    [_operation_error release];
    dispatch_release(_completed);
    [super dealloc];
}

@end

static NSString *timeout_error_message(int64_t timeout_nanoseconds) {
    double timeout_seconds = (double)timeout_nanoseconds / (double)NSEC_PER_SEC;
    return [NSString stringWithFormat:@"NSWorkspace.recycleURLs: timed out after %.3f seconds",
                                      timeout_seconds];
}

char *mac_cleanup_recycle_paths(const char *paths_json, int64_t timeout_nanoseconds) {
    @autoreleasepool {
        if (paths_json == NULL) {
            return copy_response(@[], @"NSWorkspace.recycleURLs: missing request");
        }

        NSData *input = [NSData dataWithBytes:paths_json length:strlen(paths_json)];
        NSError *decode_error = nil;
        id decoded = [NSJSONSerialization JSONObjectWithData:input options:0 error:&decode_error];
        if (![decoded isKindOfClass:[NSArray class]]) {
            NSString *message = decode_error.localizedDescription ?: @"request is not an array";
            return copy_response(@[], [@"NSWorkspace.recycleURLs: " stringByAppendingString:message]);
        }

        NSArray<NSString *> *paths = decoded;
        for (id path in paths) {
            if (![path isKindOfClass:[NSString class]]) {
                return copy_response(@[], @"NSWorkspace.recycleURLs: request contains a non-string path");
            }
        }

        MacCleanupRecycleOperation *operation = [[MacCleanupRecycleOperation alloc] init];

        dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
            @autoreleasepool {
                NSMutableArray<NSURL *> *urls = [NSMutableArray arrayWithCapacity:paths.count];
                for (NSString *path in paths) {
                    [urls addObject:[NSURL fileURLWithPath:path]];
                }

                [[NSWorkspace sharedWorkspace]
                    recycleURLs:urls
                    completionHandler:^(NSDictionary<NSURL *, NSURL *> *new_urls, NSError *error) {
                        NSMutableArray<NSString *> *completed_paths =
                            [NSMutableArray arrayWithCapacity:paths.count];
                        for (NSUInteger i = 0; i < urls.count; i++) {
                            if ([new_urls objectForKey:urls[i]] != nil) {
                                [completed_paths addObject:paths[i]];
                            }
                        }
                        [operation completeWithMovedPaths:completed_paths error:error];
                    }];
            }
        });

        if (![operation waitForTimeoutNanoseconds:timeout_nanoseconds]) {
            char *response = copy_response(@[], timeout_error_message(timeout_nanoseconds));
            [operation release];
            return response;
        }

        char *response = [operation copyResponse];
        [operation release];
        return response;
    }
}
