//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>

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

char *mac_cleanup_recycle_paths(const char *paths_json) {
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

        dispatch_semaphore_t completed = dispatch_semaphore_create(0);
        __block NSArray<NSString *> *moved_paths = nil;
        __block NSError *operation_error = nil;

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
                        moved_paths = [completed_paths copy];
                        operation_error = [error copy];
                        dispatch_semaphore_signal(completed);
                    }];
            }
        });

        dispatch_semaphore_wait(completed, DISPATCH_TIME_FOREVER);
        char *response = copy_response(moved_paths, operation_error_message(operation_error));
        [moved_paths release];
        [operation_error release];
        dispatch_release(completed);
        return response;
    }
}
