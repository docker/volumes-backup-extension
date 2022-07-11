import { useState } from "react";
import { IconButton, Snackbar } from "@mui/material";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";

export default function CopyButton({ ...props}) {
    const [open, setOpen] = useState(false);

    const handleClick = () => {
        setOpen(true);
        console.log(props.content);
        navigator.clipboard.writeText(props.content);
    };

    return (
        <>
            <IconButton
                onClick={handleClick}
                color="primary"
                style={{ float: "right", marginTop: "14px" }}
            >
                <ContentCopyIcon />
            </IconButton>
            <Snackbar
                message="Copied to clipboard"
                anchorOrigin={{ vertical: "top", horizontal: "center" }}
                autoHideDuration={2000}
                onClose={() => setOpen(false)}
                open={open}
            />
        </>
    );
};

