<?php
header("Content-Type: text/plain; charset=utf-8");

function dump_line($label, $value) {
    echo $label, ": ";
    if ($value === null) {
        echo "(null)";
    } elseif (is_bool($value)) {
        echo $value ? "true" : "false";
    } else {
        echo $value;
    }
    echo "\n";
}

echo "Goro PHP Server Index\n";
echo "=====================\n\n";
echo "Compact phpinfo()-style summary for goro.\n";
echo "The real phpinfo() function is not implemented yet.\n\n";

dump_line("PHP_VERSION", PHP_VERSION);
dump_line("PHP_SAPI", php_sapi_name());
dump_line("PHP_UNAME", php_uname("a"));
dump_line("CWD", getcwd());
dump_line("REQUEST_METHOD", isset($_SERVER["REQUEST_METHOD"]) ? $_SERVER["REQUEST_METHOD"] : "");
dump_line("REQUEST_URI", isset($_SERVER["REQUEST_URI"]) ? $_SERVER["REQUEST_URI"] : "");
dump_line("DOCUMENT_ROOT", isset($_SERVER["DOCUMENT_ROOT"]) ? $_SERVER["DOCUMENT_ROOT"] : "");
dump_line("SCRIPT_NAME", isset($_SERVER["SCRIPT_NAME"]) ? $_SERVER["SCRIPT_NAME"] : "");
dump_line("SCRIPT_FILENAME", isset($_SERVER["SCRIPT_FILENAME"]) ? $_SERVER["SCRIPT_FILENAME"] : "");
dump_line("PATH_INFO", isset($_SERVER["PATH_INFO"]) ? $_SERVER["PATH_INFO"] : "");
dump_line("REMOTE_ADDR", isset($_SERVER["REMOTE_ADDR"]) ? $_SERVER["REMOTE_ADDR"] : "");
dump_line("SERVER_ADDR", isset($_SERVER["SERVER_ADDR"]) ? $_SERVER["SERVER_ADDR"] : "");

echo "\nConfig\n";
echo "------\n";
$config_keys = array(
    "cfg_file_path",
    "default_charset",
    "display_errors",
    "file_uploads",
    "html_errors",
    "include_path",
    "max_execution_time",
    "memory_limit",
    "output_buffering",
    "register_argc_argv",
    "variables_order",
);
foreach ($config_keys as $key) {
    dump_line($key, get_cfg_var($key));
}

echo "\nFunctions\n";
echo "---------\n";
$functions = array(
    "phpinfo",
    "phpversion",
    "php_sapi_name",
    "get_cfg_var",
    "header",
    "headers_list",
    "http_response_code",
    "print_r",
    "var_dump",
);
foreach ($functions as $name) {
    dump_line($name, function_exists($name));
}

echo "\nExtensions\n";
echo "----------\n";
$extensions = array(
    "standard",
);
foreach ($extensions as $name) {
    dump_line($name, extension_loaded($name));
}

echo "\n_SERVER (" . count($_SERVER) . ")\n";
echo "----------------\n";
print_r($_SERVER);

echo "\n_ENV (" . count($_ENV) . ")\n";
echo "-------------\n";
print_r($_ENV);
