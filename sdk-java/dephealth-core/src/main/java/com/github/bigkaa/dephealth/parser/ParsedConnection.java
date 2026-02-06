package com.github.bigkaa.dephealth.parser;

import com.github.bigkaa.dephealth.DependencyType;

/**
 * Результат парсинга URL/connection string: хост, порт, тип зависимости.
 */
public record ParsedConnection(String host, String port, DependencyType type) {

    @Override
    public String toString() {
        return type.label() + "://" + host + ":" + port;
    }
}
