import os
import re

base = 'd:/ToenToen/webapi-go/internal/domain'
for root, _, files in os.walk(base):
    for f in files:
        if f.endswith('.go'):
            path = os.path.join(root, f)
            with open(path, 'r', encoding='utf-8') as file:
                content = file.read()
            
            new_content = content.replace('getRequesterInfo(c)', 'utils.GetRequesterInfo(c)')
            
            # Remove the function definition from product_handler.go
            if f == 'product_handler.go':
                new_content = re.sub(r'// Helper function.*?func getRequesterInfo.*?return requesterID, requesterRole, nil\s*\}', '', new_content, flags=re.DOTALL)
            
            # Make sure utils package is imported if utils.GetRequesterInfo is used
            if 'utils.GetRequesterInfo' in new_content and '"toentoen-backend/pkg/utils"' not in new_content:
                new_content = re.sub(r'(import \()', r'\1\n\t"toentoen-backend/pkg/utils"', new_content)
                
            if content != new_content:
                with open(path, 'w', encoding='utf-8') as file:
                    file.write(new_content)
                print(f'Updated {path}')
