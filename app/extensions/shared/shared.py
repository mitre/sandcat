import re

from plugins.sandcat.app.utility.base_extension import Extension


EXPORT_PLACEHOLDER = '// ADDITIONAL EXPORTS PLACEHOLDER'


def load():
    return Shared()


class Shared(Extension):
    def __init__(self):
        super().__init__(
            [('shared.go', 'shared'),],
            file_hooks={'shared.go': self.hook_set_additional_exports}
        )
        self.dependencies = []
        self.additional_exports = []

    async def copy_module_files(self, base_dir, headers=None):
        # Check if additional export functions are requested
        if headers and 'additional_exports' in headers:
            for export in headers.get('additional_exports', '').split(','):
                if export:
                    self.additional_exports.append(export.strip())

        return await super().copy_module_files(base_dir, headers=headers)

    async def hook_set_additional_exports(self, original_data):
        """Will add additional export functions in shared.go to run agent."""
        if self.additional_exports:
            export_text = ''
            for export_func in self.additional_exports:
                sanitized = sanitize_export_func(export_func)
                export_text += f'//export {sanitized}\nfunc {sanitized}() {{\n    VoidFunc()\n}}\n\n'
            
            self.additional_exports.clear()
            if export_text:
                return original_data.replace(EXPORT_PLACEHOLDER, export_text)
        return original_data

    @staticmethod
    def sanitize_export_func(export_func):
        return re.sub('[^0-9a-zA-Z_]+', '_', export_func)
