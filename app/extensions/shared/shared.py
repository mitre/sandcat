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
        if headers:
            for export in headers.get('additional_exports', '').split(','):
                self.additional_exports.append(export.strip())

        return await super().copy_module_files(base_dir, headers=headers)

    def remove_module_files(self, base_dir):
        self.additional_exports.clear()
        super().remove_module_files(base_dir)

    async def hook_set_additional_exports(self, original_data):
        """Will add additional export functions in shared.go to run agent."""
        if self.additional_exports:
            export_text = ''
            for export_func in self.additional_exports:
                export_text += f'//export {export_func}\nfunc {export_func}() {{\n    VoidFunc()\n}}\n\n'

            if export_text:
                return re.sub(EXPORT_PLACEHOLDER, export_text, original_data, count=1)
        return original_data
