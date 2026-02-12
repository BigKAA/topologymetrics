package biz.kryukov.dev.dephealth.parser;

import biz.kryukov.dev.dephealth.DependencyType;

/**
 * Result of URL/connection string parsing: host, port, dependency type.
 */
public record ParsedConnection(String host, String port, DependencyType type) {

    @Override
    public String toString() {
        return type.label() + "://" + host + ":" + port;
    }
}
