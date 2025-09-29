import js from "@eslint/js";
import globals from "globals";
import {defineConfig} from "eslint/config";
import {jsdoc} from 'eslint-plugin-jsdoc';

export default defineConfig([
    {
        files: ["src/**/*.js"],
        plugins: {js},
        extends: ["js/recommended"],
        languageOptions: {
            globals: globals.browser
        },
    },
    jsdoc({
        files: ["src/**/*.js"],
        config: 'flat/recommended',
        rules: {
            // These errors stem from the plugin being unable to understand the full TS syntax being used.
            // And since TS check all this already, there's no reason for this plugin to do it at all.
            "jsdoc/no-undefined-types": 0,
            // TODO: Enable later...
            "jsdoc/require-param-description": 0,
            "jsdoc/require-returns-description": 0,
            "jsdoc/require-returns-type": 0,
            "jsdoc/require-property-description": 0,
        },
    }),
]);
